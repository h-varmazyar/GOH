package xrayclient

import (
	"net/http"
	"time"
)

// Transport returns an isolated HTTP transport whose connections use Xray.
// It never changes http.DefaultTransport or process-wide proxy settings.
func (c *Client) Transport(base *http.Transport) *http.Transport {
	if base == nil {
		base = http.DefaultTransport.(*http.Transport)
	}
	transport := base.Clone()
	transport.Proxy = nil
	transport.DialContext = c.DialContext
	transport.DialTLSContext = nil
	transport.ForceAttemptHTTP2 = true
	return transport
}

// HTTPClient returns an opt-in HTTP client backed by this Xray instance.
func (c *Client) HTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: c.Transport(nil),
		Timeout:   timeout,
	}
}
