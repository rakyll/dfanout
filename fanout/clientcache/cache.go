package clientcache

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"sync"
	"time"

	pb "github.com/dfanout/dfanout/proto"
)

const defaultTimeout = 15 * time.Second

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

	client, err := c.RegisterHTTPClient(fanout, e)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (c *Cache) key(fanout string, endpointName string) string {
	return fanout + ":" + endpointName // TODO: Make ":" a reserved character
}

func (c *Cache) RegisterHTTPClient(fanout string, e *pb.Endpoint) (*http.Client, error) {
	httpEndpoint := e.Destination.(*pb.Endpoint_HttpEndpoint).HttpEndpoint

	tr := &http.Transport{}
	if tlsConfig := httpEndpoint.TlsConfig; tlsConfig != nil {
		config := &tls.Config{}
		config.ServerName = tlsConfig.ServerName
		config.InsecureSkipVerify = tlsConfig.InsecureSkipVerify
		if tlsConfig.CaPem != nil {
			roots := x509.NewCertPool()
			ok := roots.AppendCertsFromPEM(tlsConfig.CaPem)
			if !ok {
				return nil, fmt.Errorf("failed to parse root certificate for %q, %q", fanout, e.Name)
			}
			config.RootCAs = roots
			config.ClientAuth = tls.RequireAndVerifyClientCert
		}
		if tlsConfig.CaPem != nil && tlsConfig.KeyPem != nil {
			cert, err := tls.X509KeyPair(tlsConfig.CaPem, tlsConfig.KeyPem)
			if err != nil {
				return nil, fmt.Errorf("failed to load X509 key value pair for %q, %q: %w", fanout, e.Name, err)
			}
			config.Certificates = []tls.Certificate{cert}
		}
		tr.TLSClientConfig = config
	}

	timeout := defaultTimeout
	if tms := httpEndpoint.TimeoutMs; tms > 0 {
		timeout = time.Duration(tms) * time.Millisecond
	}

	c.Lock()
	defer c.Unlock()

	client := &http.Client{Transport: tr}
	c.httpClients[c.key(fanout, e.Name)] = client

	return &http.Client{
		Transport: tr,
		Timeout:   timeout,
	}, nil
}
