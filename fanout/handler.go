package fanout

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/dfanout/dfanout/clientcache"
	"github.com/dfanout/dfanout/debug"
	pb "github.com/dfanout/dfanout/proto"
	"github.com/gorilla/mux"
)

type Handler struct {
	ClientCache *clientcache.Cache
	FanoutCache *Cache

	MaxEndpointTimeout time.Duration
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fanout := vars["name"]
	log.Printf("Serving fanout = %q", fanout)

	if fanout == "" {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "missing fanout name")
		return
	}

	// Reject circular calls. Fanouts can call into other fanouts, but should
	// reject triggering themselves to avoid circular calls.
	if h := r.Header.Values(circularRequestDetectionHeader); len(h) > 0 && h[0] == fanout {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "rejected circular call")
		return
	}

	endpoints, err := h.FanoutCache.Fanout(r.Context(), fanout)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "cannot retrieve the fanout: %v", err)
		return
	}

	if r.URL.Query().Has("debug") {
		debug.NewHandler(fanout, endpoints).ServeHTTP(w, r)
		return
	}

	worker := &Worker{
		fanout:             fanout,
		endpoints:          endpoints,
		clientCache:        h.ClientCache,
		maxEndpointTimeout: h.MaxEndpointTimeout,
	}
	worker.Wait(w, r)
}

type Worker struct {
	fanout             string
	clientCache        *clientcache.Cache
	endpoints          []*pb.Endpoint
	maxEndpointTimeout time.Duration

	resp *workerResponse // mutated by the primary response
}

func (worker *Worker) Wait(w http.ResponseWriter, r *http.Request) {
	// TODO: Set a cap on maximum number of concurrent outgoing requests.
	// Fail when primary endpoint fails.
	var wg sync.WaitGroup
	wg.Add(len(worker.endpoints))

	for _, endpoint := range worker.endpoints {
		go func(e *pb.Endpoint) {
			defer wg.Done()

			worker.do(r, worker.fanout, e)
		}(endpoint)
	}
	wg.Wait()

	if worker.resp == nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "timed out with no response")
		return
	}

	if err := worker.resp.Copy(w); err != nil {
		fmt.Fprintf(w, "failed to serve body: %v", err)
		return
	}
}

func (worker *Worker) do(r *http.Request, fanout string, endpoint *pb.Endpoint) {
	log.Printf("Making a request to = %q/%q", fanout, endpoint.Name)
	defer log.Printf("Done with a request to = %q/%q", fanout, endpoint.Name)

	httpEndpoint := endpoint.Endpoint.(*pb.Endpoint_HttpEndpoint).HttpEndpoint

	timeout := worker.maxEndpointTimeout
	if n := httpEndpoint.TimeoutMs; n > 0 {
		// TODO: Enforce the global max timeout?
		timeout = time.Duration(n) * time.Millisecond
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	r = r.Clone(ctx)
	url, _ := url.Parse(httpEndpoint.Url)
	r.RequestURI = ""
	r.URL = url
	r.Method = httpEndpoint.Method

	if len(httpEndpoint.Headers) > 0 {
		for _, h := range httpEndpoint.Headers {
			for _, v := range h.Values {
				r.Header.Add(h.Key, v)
			}
		}
	}

	// Set a header to avoid the fanout triggering itself.
	// Don't remove this header.
	r.Header.Set(circularRequestDetectionHeader, fanout)

	client, err := worker.clientCache.HTTPClient(fanout, endpoint)
	if err != nil {
		log.Printf("Failed to create a client for %q/%q; err = %q", fanout, endpoint.Name, err)
		return
	}

	resp, err := client.Do(r)
	if err != nil {
		log.Printf("Failed a request to = %q/%q; err = %q", fanout, endpoint.Name, err)
		return
	}
	if !endpoint.Primary {
		resp.Body.Close() // discard the response
		return
	}
	worker.resp = &workerResponse{
		code:   resp.StatusCode,
		header: resp.Header,
		body:   resp.Body,
	}
}

type workerResponse struct {
	code   int
	header http.Header
	body   io.ReadCloser
}

// Copy copies the worker's primary endpoint response
// to the fanout handler's response.
func (r *workerResponse) Copy(w http.ResponseWriter) error {
	w.WriteHeader(r.code)
	for k, vv := range r.header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	if r.body != nil {
		defer r.body.Close()
		_, err := io.Copy(w, r.body)
		return err
	}
	return nil
}

const circularRequestDetectionHeader = "DFanout-Fanout"
