package xrayclient

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// ConfigForOutbound wraps one standard Xray outbound object in a minimal config.
// The outbound can use vmess, vless, socks, hysteria, and XHTTP stream settings.
func ConfigForOutbound(outbound json.RawMessage) ([]byte, error) {
	var object map[string]any
	if err := json.Unmarshal(outbound, &object); err != nil {
		return nil, fmt.Errorf("decode Xray outbound: %w", err)
	}
	if object["protocol"] == nil {
		return nil, fmt.Errorf("Xray outbound has no protocol")
	}

	var config bytes.Buffer
	config.WriteString(`{"log":{"loglevel":"warning"},"outbounds":[`)
	config.Write(outbound)
	config.WriteString(`]}`)
	return config.Bytes(), nil
}
