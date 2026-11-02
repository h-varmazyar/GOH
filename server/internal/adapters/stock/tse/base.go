package tse

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	baseURL                        = "https://cdn.tsetmc.com/api"
	fetchInstrumentInfoURL         = "/Instrument/GetInstrumentInfo/%v"
	fetchInstrumentStateHistoryURL = "/MarketData/GetInstrumentStateAll/%v"
	fetchClosingPriceHistoryURL    = "/ClosingPrice/GetClosingPriceDailyList/%v/0"
	fetchCurrentClosingPriceURL    = "/ClosingPrice/GetClosingPriceInfo/%v"
	fetchClientTypeHistory         = "/ClientType/GetClientTypeHistory/%v"
	fetchCurrentClientTypeURL      = "/ClientType/GetClientType/%v/1/0"
	fetchTradeHistoryURL           = "/Trade/GetTradeHistory/%v/%v/false"
	fetchInstrumentSearchURL       = "/Instrument/GetInstrumentSearch/%v"
	fetchStaticDataURL             = "/StaticData/GetStaticData"
)

const userAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36"

const marketWatchURL = "https://old.tsetmc.com/tsev2/data/MarketWatchInit.aspx"

func (a *Adapter) fetchJSON(ctx context.Context, path string, target any) error {
	return a.getJSON(ctx, baseURL+path, target)
}

func (a *Adapter) getJSON(ctx context.Context, requestURL string, target any) error {
	body, err := a.get(ctx, requestURL, "application/json, text/plain, */*")
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("decode response of %s: %w", requestURL, err)
	}
	return nil
}

// get requests an absolute URL and retries network failures and 5xx responses with backoff.
func (a *Adapter) get(ctx context.Context, requestURL, accept string) ([]byte, error) {
	backoff := [...]time.Duration{30 * time.Second, time.Minute, 2 * time.Minute, 3 * time.Minute, 5 * time.Minute}
	for attempt := 0; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", accept)
		req.Header.Set("User-Agent", userAgent)
		resp, err := a.httlClient.Do(req)
		retry := false
		var requestErr error
		if err != nil {
			requestErr = err
			retry = ctx.Err() == nil
		} else if resp.StatusCode >= http.StatusInternalServerError && resp.StatusCode < 600 {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			requestErr = fmt.Errorf("%s returned status %d: %s", requestURL, resp.StatusCode, strings.TrimSpace(string(body)))
			retry = true
			_ = resp.Body.Close()
		} else if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			_ = resp.Body.Close()
			return nil, fmt.Errorf("%s returned status %d: %s", requestURL, resp.StatusCode, strings.TrimSpace(string(body)))
		} else {
			body, readErr := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if readErr == nil {
				return body, nil
			}
			requestErr = readErr
			retry = ctx.Err() == nil
		}
		if !retry || attempt >= len(backoff) {
			return nil, fmt.Errorf("request %s failed after %d attempts: %w", requestURL, attempt+1, requestErr)
		}
		timer := time.NewTimer(backoff[attempt])
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func trimOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}

func parseTSEDate(value int64) (time.Time, error) {
	date, err := time.ParseInLocation("20060102", strconv.FormatInt(value, 10), time.UTC)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid YYYYMMDD value %d: %w", value, err)
	}
	return date, nil
}
