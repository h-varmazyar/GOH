package tse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/h-varmazyar/gatec/internal/consts"
)

type InstrumentInfo struct {
	InsCode string
	Symbol  string
	Name    string
	Nav     *float64
	EPS     *float64
}

type InstrumentStateHistory struct {
	ID               int64
	Date             time.Time
	Time             int64
	InsCode          string
	Symbol           *string
	Name             *string
	Code             string
	RealTime         int64
	UnderSupervision int64
	Title            string
}

type instrumentInfoResponse struct {
	InstrumentInfo struct {
		InsCode string   `json:"insCode"`
		Symbol  string   `json:"lVal18AFC"`
		Name    string   `json:"lVal30"`
		Nav     *float64 `json:"nav"`
		EPS     *struct {
			EstimatedEPS *string `json:"estimatedEPS"`
		} `json:"eps"`
	} `json:"instrumentInfo"`
}

type instrumentStateResponse struct {
	InstrumentState []struct {
		ID               int64   `json:"idn"`
		Date             int64   `json:"dEven"`
		Time             int64   `json:"hEven"`
		InsCode          string  `json:"insCode"`
		Symbol           *string `json:"lVal18AFC"`
		Name             *string `json:"lVal30"`
		Code             string  `json:"cEtaval"`
		RealTime         int64   `json:"realHeven"`
		UnderSupervision int64   `json:"underSupervision"`
		Title            string  `json:"cEtavalTitle"`
	} `json:"instrumentState"`
}

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

func (a *Adapter) FetchInstrumentStateHistory(ctx context.Context, insCode string) ([]InstrumentStateHistory, error) {
	var r instrumentStateResponse
	if err := a.fetchJSON(ctx, fmt.Sprintf(fetchInstrumentStateHistoryURL, url.PathEscape(insCode)), &r); err != nil {
		return nil, err
	}
	result := make([]InstrumentStateHistory, 0, len(r.InstrumentState))
	for _, item := range r.InstrumentState {
		date, err := parseTSEDate(item.Date)
		if err != nil {
			return nil, err
		}
		result = append(result,
			InstrumentStateHistory{
				ID:               item.ID,
				Date:             date,
				Time:             item.Time,
				InsCode:          item.InsCode,
				Symbol:           trimOptionalString(item.Symbol),
				Name:             trimOptionalString(item.Name),
				Code:             strings.TrimSpace(item.Code),
				RealTime:         item.RealTime,
				UnderSupervision: item.UnderSupervision,
				Title:            strings.TrimSpace(item.Title),
			},
		)
	}
	return result, nil
}

func (a *Adapter) FetchInstrumentInfo(ctx context.Context, insCode string) (*InstrumentInfo, error) {
	var r instrumentInfoResponse
	if err := a.fetchJSON(ctx, fmt.Sprintf(fetchInstrumentInfoURL, url.PathEscape(insCode)), &r); err != nil {
		return nil, err
	}
	if r.InstrumentInfo.InsCode == "" {
		return nil, fmt.Errorf("instrument info response has no insCode")
	}
	result := &InstrumentInfo{
		InsCode: r.InstrumentInfo.InsCode,
		Symbol:  strings.TrimSpace(r.InstrumentInfo.Symbol),
		Name:    strings.TrimSpace(r.InstrumentInfo.Name),
		Nav:     r.InstrumentInfo.Nav,
	}
	if r.InstrumentInfo.EPS != nil && r.InstrumentInfo.EPS.EstimatedEPS != nil {
		if value, err := strconv.ParseFloat(strings.TrimSpace(*r.InstrumentInfo.EPS.EstimatedEPS), 64); err == nil {
			result.EPS = &value
		}
	}
	return result, nil
}

// func (n *Adapter) fetchInstruments(ctx context.Context) ([]Instrument, error) {
// 	pageSize := 100
// 	result := make([]Instrument, 0)
// 	for page := 1; ; page++ {
// 		endpoint, err := url.Parse(instrumentSearchURL)
// 		if err != nil {
// 			return nil, fmt.Errorf("parse TSE instrument search URL: %w", err)
// 		}
// 		query := endpoint.Query()
// 		query.Set("pageSize", strconv.Itoa(pageSize))
// 		query.Set("pageNumber", strconv.Itoa(page))
// 		endpoint.RawQuery = query.Encode()
// 		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
// 		if err != nil {
// 			return nil, fmt.Errorf("create TSE instrument search request: %w", err)
// 		}
// 		resp, err := n.httlClient.Do(req)
// 		if err != nil {
// 			return nil, fmt.Errorf("request TSE instrument search: %w", err)
// 		}
// 		body, readErr := io.ReadAll(resp.Body)
// 		resp.Body.Close()
// 		if readErr != nil {
// 			return nil, fmt.Errorf("read TSE instrument search response: %w", readErr)
// 		}
// 		if resp.StatusCode != http.StatusOK {
// 			return nil, fmt.Errorf("TSE instrument search returned status %d", resp.StatusCode)
// 		}
// 		var response instrumentCompareResponse
// 		if err := json.Unmarshal(body, &response); err != nil {
// 			return nil, fmt.Errorf("decode TSE instrument search response: %w", err)
// 		}
// 		pageItems := 0
// 		for _, item := range response.Items {
// 			for _, instrument := range item.Instruments {
// 				status := consts.InstrumentStatusDisable
// 				if instrument.Status == 1 {
// 					status = consts.InstrumentStatusEnable
// 				}
// 				result = append(result, Instrument{Name: instrument.Name, Label: instrument.Symbol, Type: strconv.Itoa(instrument.Type), ID: instrument.ID, Group: strings.TrimSpace(instrument.Group), SubGroup: strings.TrimSpace(instrument.SubGroup), Status: status})
// 				pageItems++
// 			}
// 		}
// 		if pageItems == 0 || pageItems < pageSize {
// 			break
// 		}
// 	}
// 	return result, nil
// }
