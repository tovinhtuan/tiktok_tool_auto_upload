package infrastructure

import (
	"crypto/tls"
	"net/http"
	"time"

	"auto_upload_tiktok/config"
)

// HTTPClient provides a high-performance HTTP client with connection pooling
type HTTPClient struct {
	client *http.Client
	config *config.Config
}

// NewHTTPClient creates a new optimized HTTP client for I/O bound operations
func NewHTTPClient(cfg *config.Config) *HTTPClient {
	transport := &http.Transport{
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxConnsPerHost,
		MaxConnsPerHost:     cfg.MaxConnsPerHost, // Allow more concurrent connections per host
		IdleConnTimeout:     90 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second, // Timeout for reading response headers
		ExpectContinueTimeout:  1 * time.Second,  // Timeout for 100-continue
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
		DisableKeepAlives: false,
		ForceAttemptHTTP2: true, // HTTP/2 for better multiplexing
		WriteBufferSize:   64 * 1024, // 64KB write buffer
		ReadBufferSize:    64 * 1024, // 64KB read buffer
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   cfg.HTTPClientTimeout,
	}

	return &HTTPClient{
		client: client,
		config: cfg,
	}
}

// Get performs a GET request
func (c *HTTPClient) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return c.client.Do(req)
}

// Do performs a custom HTTP request
func (c *HTTPClient) Do(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}

// GetClient returns the underlying HTTP client
func (c *HTTPClient) GetClient() *http.Client {
	return c.client
}

