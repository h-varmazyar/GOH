package tse

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const historyBaseURL = "https://cdn.tsetmc.com/api"

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
type ClosingPriceHistory struct {
	ClosingPrice
	LastChange float64
	Closing    float64
}

type ClosingPrice struct {
	Date       time.Time
	Yesterday  float64
	First      float64
	Last       float64
	Close      float64
	Low        float64
	High       float64
	Volume     int64
	Value      int64
	TradeCount int64
}

type ClientType struct {
	Date                    time.Time
	BuyIndividualVolume     int64
	BuyIndividualValue      int64
	BuyIndividualCount      int64
	BuyInstitutionalVolume  int64
	BuyInstitutionalValue   int64
	BuyInstitutionalCount   int64
	SellIndividualVolume    int64
	SellIndividualValue     int64
	SellIndividualCount     int64
	SellInstitutionalVolume int64
	SellInstitutionalValue  int64
	SellInstitutionalCount  int64
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
type closingPriceResponse struct {
	ClosingPriceDaily []struct {
		Date           int64   `json:"dEven"`
		TradeCount     float64 `json:"zTotTran"`
		Volume         float64 `json:"qTotTran5J"`
		Value          float64 `json:"qTotCap"`
		PriceYesterday float64 `json:"priceYesterday"`
		PriceFirst     float64 `json:"priceFirst"`
		Last           float64 `json:"pDrCotVal"`
		LastChange     float64 `json:"priceChange"`
		Closing        float64 `json:"pClosing"`
		Low            float64 `json:"priceMin"`
		High           float64 `json:"priceMax"`
	} `json:"closingPriceDaily"`
}

type currentClosingResponse struct {
	ClosingPriceInfo struct {
		InstrumentState struct {
			Code string `json:"cEtaval"`
		} `json:"instrumentState"`
		FinalDate  int64   `json:"finalLastDate"`
		Date       int64   `json:"dEven"`
		TradeCount float64 `json:"zTotTran"`
		Volume     float64 `json:"qTotTran5J"`
		Value      float64 `json:"qTotCap"`
		Yesterday  float64 `json:"priceYesterday"`
		First      float64 `json:"priceFirst"`
		Last       float64 `json:"pDrCotVal"`
		Close      float64 `json:"pClosing"`
		Low        float64 `json:"priceMin"`
		High       float64 `json:"priceMax"`
	} `json:"closingPriceInfo"`
}
type currentClientResponse struct {
	ClientType map[string]float64 `json:"clientType"`
}
type clientHistoryResponse struct {
	ClientType []struct {
		RecDate     int64   `json:"recDate"`
		BuyIVolume  float64 `json:"buy_I_Volume"`
		BuyNVolume  float64 `json:"buy_N_Volume"`
		BuyIValue   float64 `json:"buy_I_Value"`
		BuyNValue   float64 `json:"buy_N_Value"`
		BuyICount   float64 `json:"buy_I_Count"`
		BuyNCount   float64 `json:"buy_N_Count"`
		SellIVolume float64 `json:"sell_I_Volume"`
		SellNVolume float64 `json:"sell_N_Volume"`
		SellIValue  float64 `json:"sell_I_Value"`
		SellNValue  float64 `json:"sell_N_Value"`
		SellICount  float64 `json:"sell_I_Count"`
		SellNCount  float64 `json:"sell_N_Count"`
	}
}

func (a *Adapter) FetchInstrumentInfo(ctx context.Context, insCode string) (*InstrumentInfo, error) {
	var r instrumentInfoResponse
	if err := a.fetchJSON(ctx, "/Instrument/GetInstrumentInfo/"+url.PathEscape(insCode), &r); err != nil {
		return nil, err
	}
	if r.InstrumentInfo.InsCode == "" {
		return nil, fmt.Errorf("instrument info response has no insCode")
	}
	result := &InstrumentInfo{InsCode: r.InstrumentInfo.InsCode, Symbol: strings.TrimSpace(r.InstrumentInfo.Symbol), Name: strings.TrimSpace(r.InstrumentInfo.Name), Nav: r.InstrumentInfo.Nav}
	if r.InstrumentInfo.EPS != nil && r.InstrumentInfo.EPS.EstimatedEPS != nil {
		if value, err := strconv.ParseFloat(strings.TrimSpace(*r.InstrumentInfo.EPS.EstimatedEPS), 64); err == nil {
			result.EPS = &value
		}
	}
	return result, nil
}

func (a *Adapter) FetchInstrumentStateHistory(ctx context.Context, insCode string) ([]InstrumentStateHistory, error) {
	var r instrumentStateResponse
	if err := a.fetchJSON(ctx, "/MarketData/GetInstrumentStateAll/"+url.PathEscape(insCode), &r); err != nil {
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

func (a *Adapter) FetchClosingPriceHistory(ctx context.Context, insCode string) ([]ClosingPriceHistory, error) {
	var r closingPriceResponse
	if err := a.fetchJSON(ctx, "/ClosingPrice/GetClosingPriceDailyList/"+url.PathEscape(insCode)+"/0", &r); err != nil {
		return nil, err
	}
	result := make([]ClosingPriceHistory, 0, len(r.ClosingPriceDaily))
	for _, item := range r.ClosingPriceDaily {
		date, err := parseTSEDate(item.Date)
		if err != nil {
			return nil, err
		}
		result = append(result,
			ClosingPriceHistory{
				ClosingPrice: ClosingPrice{
					Date:       date,
					TradeCount: int64(item.TradeCount),
					Volume:     int64(item.Volume),
					Value:      int64(item.Value),
					Yesterday:  item.PriceYesterday,
					First:      item.PriceFirst,
					Last:       item.Last,
					Low:        item.Low,
					High:       item.High,
				},
				LastChange: item.LastChange,
				Closing:    item.Closing,
			},
		)
	}
	return result, nil
}
func (a *Adapter) FetchClientTypeHistory(ctx context.Context, insCode string) ([]ClientType, error) {
	var r clientHistoryResponse
	if err := a.fetchJSON(ctx, "/ClientType/GetClientTypeHistory/"+url.PathEscape(insCode), &r); err != nil {
		return nil, err
	}
	result := make([]ClientType, 0, len(r.ClientType))
	for _, item := range r.ClientType {
		date, err := parseTSEDate(item.RecDate)
		if err != nil {
			return nil, err
		}
		result = append(result,
			ClientType{
				Date:                    date,
				BuyIndividualVolume:     int64(item.BuyIVolume),
				BuyIndividualValue:      int64(item.BuyIValue),
				BuyIndividualCount:      int64(item.BuyICount),
				BuyInstitutionalVolume:  int64(item.BuyNVolume),
				BuyInstitutionalValue:   int64(item.BuyNValue),
				BuyInstitutionalCount:   int64(item.BuyNCount),
				SellIndividualVolume:    int64(item.SellIVolume),
				SellIndividualValue:     int64(item.SellIValue),
				SellIndividualCount:     int64(item.SellICount),
				SellInstitutionalVolume: int64(item.SellNVolume),
				SellInstitutionalValue:  int64(item.SellNValue),
				SellInstitutionalCount:  int64(item.SellNCount),
			},
		)
	}
	return result, nil
}
func (a *Adapter) FetchCurrentClosingPrice(ctx context.Context, insCode string) (*ClosingPrice, error) {
	var r currentClosingResponse
	if err := a.fetchJSON(ctx, "/ClosingPrice/GetClosingPriceInfo/"+url.PathEscape(insCode), &r); err != nil {
		return nil, err
	}
	item := r.ClosingPriceInfo
	result := &ClosingPrice{
		TradeCount: int64(item.TradeCount),
		Volume:     int64(item.Volume),
		Value:      int64(item.Value),
		Yesterday:  item.Yesterday,
		First:      item.First,
		Last:       item.Last,
		Close:      item.Close,
		Low:        item.Low,
		High:       item.High,
	}
	if item.Date != 0 {
		var err error
		result.Date, err = parseTSEDate(item.Date)
		if err != nil {
			return nil, err
		}
		// result.FinalLastDate = result.Date
		// if item.FinalDate != 0 {
		// 	result.FinalLastDate, err = parseTSEDate(item.FinalDate)
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// }
	}
	return result, nil
}
func (a *Adapter) FetchCurrentClientType(ctx context.Context, insCode string) (*ClientType, error) {
	var r currentClientResponse
	if err := a.fetchJSON(ctx, "/ClientType/GetClientType/"+url.PathEscape(insCode)+"/1/0", &r); err != nil {
		return nil, err
	}
	item := r.ClientType
	return &ClientType{
		BuyIndividualVolume:     int64(item["buy_I_Volume"]),
		BuyIndividualCount:      int64(item["buy_CountI"]),
		BuyIndividualValue:      int64(item["buy_I_Value"]),
		BuyInstitutionalVolume:  int64(item["buy_N_Volume"]),
		BuyInstitutionalCount:   int64(item["buy_CountN"]),
		BuyInstitutionalValue:   int64(item["buy_N_Value"]),
		SellIndividualVolume:    int64(item["sell_I_Volume"]),
		SellIndividualCount:     int64(item["sell_CountI"]),
		SellIndividualValue:     int64(item["sell_I_Value"]),
		SellInstitutionalVolume: int64(item["sell_N_Volume"]),
		SellInstitutionalCount:  int64(item["sell_CountN"]),
		SellInstitutionalValue:  int64(item["sell_N_Value"]),
	}, nil
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
func (a *Adapter) fetchJSON(ctx context.Context, path string, target any) error {
	requestURL := historyBaseURL + path
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
