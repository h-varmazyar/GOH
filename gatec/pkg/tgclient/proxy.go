package tgclient

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zelenin/go-tdlib/client"
)

// ProxyType represents the type of proxy
type ProxyType string

const (
	ProxyTypeMTProto ProxyType = "mtproto"
	ProxyTypeSocks5  ProxyType = "socks5"
	ProxyTypeHTTP    ProxyType = "http"
)

// ProxyConfig holds proxy configuration for a TDLib session
type ProxyConfig struct {
	Server   string
	Port     int32
	Type     ProxyType
	Secret   string // MTProto only
	Username string // SOCKS5/HTTP only
	Password string // SOCKS5/HTTP only
	HTTPOnly bool   // HTTP only
}

// toAddProxyRequest converts ProxyConfig to TDLib's AddProxyRequest
func (p *ProxyConfig) toAddProxyRequest() *client.AddProxyRequest {
	req := &client.AddProxyRequest{
		Server: p.Server,
		Port:   p.Port,
		Enable: true,
	}

	println("req type is:", p.Type)
	switch p.Type {
	case ProxyTypeMTProto:
		req.Type = &client.ProxyTypeMtproto{
			Secret: p.Secret,
		}
	case ProxyTypeSocks5:
		req.Type = &client.ProxyTypeSocks5{
			Username: p.Username,
			Password: p.Password,
		}
	case ProxyTypeHTTP:
		req.Type = &client.ProxyTypeHttp{
			Username: p.Username,
			Password: p.Password,
			HttpOnly: p.HTTPOnly,
		}
	}

	return req
}

// toTestProxyRequest converts ProxyConfig to TDLib's TestProxyRequest
func (p *ProxyConfig) toTestProxyRequest(timeout float64) *client.TestProxyRequest {
	req := &client.TestProxyRequest{
		Server:  p.Server,
		Port:    p.Port,
		DcId:    2, // Use DC2 as default for testing
		Timeout: timeout,
	}

	switch p.Type {
	case ProxyTypeMTProto:
		req.Type = &client.ProxyTypeMtproto{
			Secret: p.Secret,
		}
	case ProxyTypeSocks5:
		req.Type = &client.ProxyTypeSocks5{
			Username: p.Username,
			Password: p.Password,
		}
	case ProxyTypeHTTP:
		req.Type = &client.ProxyTypeHttp{
			Username: p.Username,
			Password: p.Password,
			HttpOnly: p.HTTPOnly,
		}
	}

	return req
}

// SetProxy adds a proxy to an existing ready session
func (m *Manager) SetProxy(sessionID string, proxy *ProxyConfig) error {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if session.client == nil {
		return fmt.Errorf("session %s has no active client", sessionID)
	}

	addedProxy, err := session.client.AddProxy(proxy.toAddProxyRequest())

	logrus.Infof("proxy added is: %v, %v", addedProxy == nil, err)

	// session.client.EnableProxy(&client.EnableProxyRequest{
	// 	ProxyId: addedProxy.Id,
	// })
	return err
}

// CheckConnection checks if a session is still connected by calling GetMe
func (m *Manager) CheckConnection(sessionID string) error {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if session.state != SessionStateReady || session.client == nil {
		return fmt.Errorf("session %s is not ready (state: %s)", sessionID, session.state)
	}

	_, err = session.client.GetMe()
	if err != nil {
		return fmt.Errorf("connection check failed: %w", err)
	}

	return nil
}

// TestProxy tests a proxy connection using a ready session's client.
// Returns nil if the proxy is reachable and functional.
func (m *Manager) TestProxy(sessionID string, proxy *ProxyConfig) error {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if session.state != SessionStateReady || session.client == nil {
		return fmt.Errorf("session %s is not ready for proxy testing", sessionID)
	}

	_, err = session.client.TestProxy(proxy.toTestProxyRequest(10.0))
	if err != nil {
		return fmt.Errorf("proxy test failed: %w", err)
	}

	return nil
}

// TestProxyStandalone tests a proxy by creating a temporary lightweight TDLib client.
// This is useful when no active sessions are available for testing.
// It creates a minimal client just for the proxy test and cleans up afterwards.
func (m *Manager) TestProxyStandalone(proxy *ProxyConfig, timeout time.Duration) error {
	// We use a single "test" session for all proxy checks to avoid resource waste.
	// If a test session already exists and is ready, reuse it.
	const testSessionID = "b02eef38-c89d-4227-a07a-2c12b9757307"

	m.mu.RLock()
	session, exists := m.sessions[testSessionID]
	m.mu.RUnlock()

	if exists && session.client != nil {
		// result, err := jsonClient.Execute(req)
		result, err := session.client.TestProxy(proxy.toTestProxyRequest(timeout.Seconds()))
		if err != nil {
			return fmt.Errorf("proxy test failed: %w", err)
		}
		if result == nil {
			return fmt.Errorf("proxy test returned nil result")
		}
		return nil
	}

	// If no ready sessions exist at all, try any ready session
	m.mu.RLock()
	for _, s := range m.sessions {
		if s.state == SessionStateReady && s.client != nil {
			m.mu.RUnlock()
			// result, err := jsonClient.Execute(req)
			result, err := s.client.TestProxy(proxy.toTestProxyRequest(timeout.Seconds()))
			if err != nil {
				return fmt.Errorf("proxy test failed: %w", err)
			}
			if result == nil {
				return fmt.Errorf("proxy test returned nil result")
			}
			fmt.Println(result)
			return nil
		}
	}
	m.mu.RUnlock()

	return fmt.Errorf("no active session available for proxy testing")
}
