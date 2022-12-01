package clientcache

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"sync"

	pb "github.com/dfanout/dfanout/proto"
)

// TODO: Turn into an LRU cache.
// TODO: Invalidate based on endpoint configuration.

type Cache struct {
	sync.RWMutex
	httpClients map[string]*http.Client
}

func New() *Cache {
	return &Cache{httpClients: make(map[string]*http.Client)}
}

func (c *Cache) HTTPClient(fanout string, e *pb.Endpoint) (*http.Client, error) {
	key := c.key(fanout, e.Name)

	c.RLock()
	client, ok := c.httpClients[key]
	c.RUnlock()

	if ok {
		return client, nil
	}

	c.Lock()
	defer c.Unlock()

	client, err := c.makeHTTPClient(fanout, e)
	if err != nil {
		return nil, err
	}
	c.httpClients[key] = client
	return client, nil
}

// Invalidate invalidates all of the client caches for the
// specified fan and endpoint.
func (c *Cache) Invalidate(fanout string, endpointName string) {
	c.Lock()
	defer c.Unlock()

	delete(c.httpClients, c.key(fanout, endpointName))
}

func (c *Cache) key(fanout string, endpointName string) string {
	return fanout + ":" + endpointName // TODO: Make ":" a reserved character
}

func (c *Cache) makeHTTPClient(fanout string, e *pb.Endpoint) (*http.Client, error) {
	httpEndpoint := e.Endpoint.(*pb.Endpoint_HttpEndpoint).HttpEndpoint

	tr := &http.Transport{}
	if endpointTLS := httpEndpoint.Tls; endpointTLS != nil {
		config := &tls.Config{}
		config.ServerName = endpointTLS.ServerName
		config.InsecureSkipVerify = endpointTLS.InsecureSkipVerify
		if endpointTLS.CaPem != nil {
			roots := x509.NewCertPool()
			ok := roots.AppendCertsFromPEM(endpointTLS.CaPem)
			if !ok {
				return nil, fmt.Errorf("failed to parse root certificate for %q, %q", fanout, e.Name)
			}
			config.RootCAs = roots
			config.ClientAuth = tls.RequireAndVerifyClientCert
		}
		if endpointTLS.CaPem != nil && endpointTLS.KeyPem != nil {
			cert, err := tls.X509KeyPair(endpointTLS.CaPem, endpointTLS.KeyPem)
			if err != nil {
				return nil, fmt.Errorf("failed to load X509 key value pair for %q, %q: %w", fanout, e.Name, err)
			}
			config.Certificates = []tls.Certificate{cert}
		}
		tr.TLSClientConfig = config
	}
	return &http.Client{Transport: tr}, nil
}
