package tabdeal

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

const giftPageMaxBodySize = 5 << 20

var (
	reGiftCodeSection = regexp.MustCompile(`(?is)<section\b[^>]*\bid\s*=\s*["']shortcode__gift-code["'][^>]*>`)
	reGiftCodeAttr    = regexp.MustCompile(`(?i)\bdata-code\s*=\s*["']([^"']*)["']`)
)

// FetchGiftCode opens a tabdeal page and extracts the gift code from its gift-code section.
func (a *Adapter) FetchGiftCode(ctx context.Context, pageURL string) (string, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36")
	httpReq.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tabdeal gift page fetch failed: status=%d url=%s", resp.StatusCode, pageURL)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, giftPageMaxBodySize))
	if err != nil {
		return "", err
	}

	return extractGiftCode(string(body))
}

func extractGiftCode(html string) (string, error) {
	section := reGiftCodeSection.FindString(html)
	if section == "" {
		return "", fmt.Errorf("gift code section not found in page")
	}
	matches := reGiftCodeAttr.FindStringSubmatch(section)
	if len(matches) < 2 || strings.TrimSpace(matches[1]) == "" {
		return "", fmt.Errorf("gift code not found in gift code section")
	}
	return strings.TrimSpace(matches[1]), nil
}
