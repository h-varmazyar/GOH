package xrayclient

import (
	"context"
	"fmt"
	"net"

	xnet "github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/core"
)

// DialContext creates a TCP connection through this Xray instance.
// Hostname resolution and outbound selection are delegated to Xray.
func (c *Client) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if network != "tcp" && network != "tcp4" && network != "tcp6" {
		return nil, fmt.Errorf("xray dial: unsupported network %q", network)
	}

	destination, err := xnet.ParseDestination("tcp:" + address)
	if err != nil {
		return nil, fmt.Errorf("xray dial: parse destination %q: %w", address, err)
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.closed || c.instance == nil {
		return nil, ErrClosed
	}

	conn, err := core.Dial(ctx, c.instance, destination)
	if err != nil {
		return nil, fmt.Errorf("xray dial %s: %w", address, err)
	}
	return conn, nil
}
