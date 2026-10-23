package tse

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/h-varmazyar/gatec/internal/consts"
)

const instrumentSearchURL = "https://cdn.tsetmc.com/api/Instrument/GetInstrumentSearch"

type instrumentCompareResponse struct {
	Items []struct {
		Instruments []struct {
			ID       string `json:"insCode"`
			Symbol   string `json:"smallSymbolName"`
			Name     string `json:"symbolFullName"`
			Group    string `json:"industryGroupCode"`
			SubGroup string `json:"industrySubCode"`
			Type     int    `json:"type"`
			Status   int    `json:"status"`
		} `json:"instruments"`
	} `json:"items"`
}

func (n *Adapter) UpdateInstruments(ctx context.Context) (markets []Instrument, err error) {
	if instruments, fetchErr := n.fetchInstruments(ctx); fetchErr == nil && len(instruments) > 0 {
		return instruments, nil
	}

	entries, err := os.ReadDir("./assets/tse/instruments")
	if err != nil {
		return nil, fmt.Errorf("read TSE instruments directory: %w", err)
	}

	instruments := make([]Instrument, 0)
	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".json" {
			continue
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		path := filepath.Join("./assets/tse/instruments", entry.Name())
		contents, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read TSE instruments file %q: %w", path, err)
		}

		var response instrumentCompareResponse
		if err := json.Unmarshal(contents, &response); err != nil {
			return nil, fmt.Errorf("decode TSE instruments file %q: %w", path, err)
		}

		for _, item := range response.Items {
			for _, instrument := range item.Instruments {
				status := consts.InstrumentStatusDisable
				if instrument.Status == 1 {
					status = consts.InstrumentStatusEnable
				}

				instruments = append(instruments, Instrument{
					Name:     instrument.Name,
					Label:    instrument.Symbol,
					Type:     strconv.Itoa(instrument.Type),
					ID:       instrument.ID,
					Group:    strings.TrimSpace(instrument.Group),
					SubGroup: strings.TrimSpace(instrument.SubGroup),
					Status:   status,
				})
			}
		}
	}

	return instruments, nil
}

func (n *Adapter) fetchInstruments(ctx context.Context) ([]Instrument, error) {
	pageSize := 100
	result := make([]Instrument, 0)
	for page := 1; ; page++ {
		endpoint, err := url.Parse(instrumentSearchURL)
		if err != nil {
			return nil, fmt.Errorf("parse TSE instrument search URL: %w", err)
		}
		query := endpoint.Query()
		query.Set("pageSize", strconv.Itoa(pageSize))
		query.Set("pageNumber", strconv.Itoa(page))
		endpoint.RawQuery = query.Encode()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("create TSE instrument search request: %w", err)
		}
		resp, err := n.httlClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request TSE instrument search: %w", err)
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read TSE instrument search response: %w", readErr)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("TSE instrument search returned status %d", resp.StatusCode)
		}
		var response instrumentCompareResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("decode TSE instrument search response: %w", err)
		}
		pageItems := 0
		for _, item := range response.Items {
			for _, instrument := range item.Instruments {
				status := consts.InstrumentStatusDisable
				if instrument.Status == 1 {
					status = consts.InstrumentStatusEnable
				}
				result = append(result, Instrument{Name: instrument.Name, Label: instrument.Symbol, Type: strconv.Itoa(instrument.Type), ID: instrument.ID, Group: strings.TrimSpace(instrument.Group), SubGroup: strings.TrimSpace(instrument.SubGroup), Status: status})
				pageItems++
			}
		}
		if pageItems == 0 || pageItems < pageSize {
			break
		}
	}
	return result, nil
}
