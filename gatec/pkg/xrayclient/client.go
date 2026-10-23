// Package xrayclient runs an embedded Xray instance for opt-in network calls.
package xrayclient

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf/serial"

	// Register only the applications needed by an outbound-only Xray instance.
	_ "github.com/xtls/xray-core/app/dispatcher"
	_ "github.com/xtls/xray-core/app/dns"
	_ "github.com/xtls/xray-core/app/log"
	_ "github.com/xtls/xray-core/app/policy"
	_ "github.com/xtls/xray-core/app/proxyman/inbound"
	_ "github.com/xtls/xray-core/app/proxyman/outbound"
	_ "github.com/xtls/xray-core/app/router"

	// Register requested outbound protocols.
	_ "github.com/xtls/xray-core/proxy/blackhole"
	_ "github.com/xtls/xray-core/proxy/freedom"
	_ "github.com/xtls/xray-core/proxy/hysteria"
	_ "github.com/xtls/xray-core/proxy/socks"
	_ "github.com/xtls/xray-core/proxy/vless/outbound"
	_ "github.com/xtls/xray-core/proxy/vmess/outbound"

	// Register transports used by the requested protocols, including XHTTP.
	_ "github.com/xtls/xray-core/transport/internet/hysteria"
	_ "github.com/xtls/xray-core/transport/internet/reality"
	_ "github.com/xtls/xray-core/transport/internet/splithttp"
	_ "github.com/xtls/xray-core/transport/internet/tcp"
	_ "github.com/xtls/xray-core/transport/internet/tls"
)

var ErrClosed = errors.New("xray client is closed")

// Client owns one long-lived in-process Xray instance.
type Client struct {
	mu       sync.RWMutex
	instance *core.Instance
	closed   bool
}

// New parses a standard Xray JSON configuration and starts it immediately.
func New(config []byte) (*Client, error) {
	instance, err := start(config)
	if err != nil {
		return nil, err
	}
	return &Client{instance: instance}, nil
}

// NewFromFile starts a client from a standard Xray JSON file.
func NewFromFile(path string) (*Client, error) {
	config, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read Xray config: %w", err)
	}
	return New(config)
}

func start(config []byte) (*core.Instance, error) {
	if len(bytes.TrimSpace(config)) == 0 {
		return nil, fmt.Errorf("Xray config is empty")
	}

	parsed, err := serial.LoadJSONConfig(bytes.NewReader(config))
	if err != nil {
		return nil, fmt.Errorf("parse Xray config: %w", err)
	}
	instance, err := core.New(parsed)
	if err != nil {
		return nil, fmt.Errorf("create Xray instance: %w", err)
	}
	if err := instance.Start(); err != nil {
		_ = instance.Close()
		return nil, fmt.Errorf("start Xray instance: %w", err)
	}
	return instance, nil
}

// Close stops Xray and releases all persistent transports and connections.
// It is safe to call Close more than once.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	if c.instance == nil {
		return nil
	}
	err := c.instance.Close()
	c.instance = nil
	if err != nil {
		return fmt.Errorf("close Xray instance: %w", err)
	}
	return nil
}
