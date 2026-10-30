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
)

const marketWatchURL = "https://old.tsetmc.com/tsev2/data/MarketWatchInit.aspx"

func (a *Adapter) fetchJSON(ctx context.Context, path string, target any) error {
	requestURL := baseURL + path
	backoff := [...]time.Duration{30 * time.Second, time.Minute, 2 * time.Minute, 3 * time.Minute, 5 * time.Minute}
	for attempt := 0; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "application/json, text/plain, */*")
		resp, err := a.httlClient.Do(req)
		retry := false
		var requestErr error
		if err != nil {
			requestErr = err
			retry = ctx.Err() == nil
		} else if resp.StatusCode >= http.StatusInternalServerError && resp.StatusCode < 600 {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			requestErr = fmt.Errorf("TSETMC returned status %d for %s: %s", resp.StatusCode, requestURL, strings.TrimSpace(string(body)))
			retry = true
			_ = resp.Body.Close()
		} else if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			_ = resp.Body.Close()
			return fmt.Errorf("TSETMC returned status %d for %s: %s", resp.StatusCode, requestURL, strings.TrimSpace(string(body)))
		} else {
			decodeErr := json.NewDecoder(resp.Body).Decode(target)
			_ = resp.Body.Close()
			if decodeErr != nil {
				return fmt.Errorf("decode TSETMC history response: %w", decodeErr)
			}
			return nil
		}
		if !retry || attempt >= len(backoff) {
			return fmt.Errorf("request %s failed after %d attempts: %w", requestURL, attempt+1, requestErr)
		}
		timer := time.NewTimer(backoff[attempt])
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
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
