package stock

import (
	"context"
	"encoding/json"

	// "errors"
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
	ESP     *float64
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
	Date       time.Time
	TradeCount int64
	Volume     int64
	Value      int64
	Yesterday  float64
	First      float64
	Last       float64
	LastChange float64
	Close      float64
	Low        float64
	High       float64
}

type ClientTypeHistory struct {
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

type instrumentStateHistoryResponse struct {
	InstrumentState []struct {
		ID               int64   `json:"idn"`
		DEven            int64   `json:"dEven"`
		HEven            int64   `json:"hEven"`
		InsCode          string  `json:"insCode"`
		Symbol           *string `json:"lVal18AFC"`
		Name             *string `json:"lVal30"`
		Code             string  `json:"cEtaval"`
		RealHEven        int64   `json:"realHeven"`
		UnderSupervision int64   `json:"underSupervision"`
		Title            string  `json:"cEtavalTitle"`
	} `json:"instrumentState"`
}

type closingPriceResponse struct {
	ClosingPriceDaily []struct {
		DEven          int64   `json:"dEven"`
		ZTotTran       float64 `json:"zTotTran"`
		QTotTran5J     float64 `json:"qTotTran5J"`
		QTotCap        float64 `json:"qTotCap"`
		PriceYesterday float64 `json:"priceYesterday"`
		PriceFirst     float64 `json:"priceFirst"`
		PDrCotVal      float64 `json:"pDrCotVal"`
		PriceChange    float64 `json:"priceChange"`
		PClosing       float64 `json:"pClosing"`
		PriceMin       float64 `json:"priceMin"`
		PriceMax       float64 `json:"priceMax"`
	} `json:"closingPriceDaily"`
}

type CurrentClosingPrice struct {
	Date                time.Time
	FinalLastDate       time.Time
	InstrumentStateCode string
	TradeCount          int64
	Volume              int64
	Value               int64
	Yesterday           float64
	First               float64
	Last                float64
	Close               float64
	Low                 float64
	High                float64
}

type CurrentClientType struct {
	BuyIndividualVolume     int64
	BuyIndividualCount      int64
	BuyIndividualValue      int64
	BuyInstitutionalVolume  int64
	BuyInstitutionalCount   int64
	BuyInstitutionalValue   int64
	SellIndividualVolume    int64
	SellIndividualCount     int64
	SellIndividualValue     int64
	SellInstitutionalVolume int64
	SellInstitutionalCount  int64
	SellInstitutionalValue  int64
}

type currentClosingPriceResponse struct {
	ClosingPriceInfo struct {
		InstrumentState struct {
			Code             string `json:"cEtaval"`
			UnderSupervision int64  `json:"underSupervision"`
		} `json:"instrumentState"`
		FinalLastDate  int64   `json:"finalLastDate"`
		DEven          int64   `json:"dEven"`
		ZTotTran       float64 `json:"zTotTran"`
		QTotTran5J     float64 `json:"qTotTran5J"`
		QTotCap        float64 `json:"qTotCap"`
		PriceYesterday float64 `json:"priceYesterday"`
		PriceFirst     float64 `json:"priceFirst"`
		PDrCotVal      float64 `json:"pDrCotVal"`
		PClosing       float64 `json:"pClosing"`
		PriceMin       float64 `json:"priceMin"`
		PriceMax       float64 `json:"priceMax"`
	} `json:"closingPriceInfo"`
}

type currentClientTypeResponse struct {
	ClientType struct {
		BuyIVolume  float64 `json:"buy_I_Volume"`
		BuyNVolume  float64 `json:"buy_N_Volume"`
		BuyICount   float64 `json:"buy_CountI"`
		BuyNCount   float64 `json:"buy_CountN"`
		BuyIValue   float64 `json:"buy_I_Value"`
		BuyNValue   float64 `json:"buy_N_Value"`
		SellIVolume float64 `json:"sell_I_Volume"`
		SellNVolume float64 `json:"sell_N_Volume"`
		SellICount  float64 `json:"sell_CountI"`
		SellNCount  float64 `json:"sell_CountN"`
		SellIValue  float64 `json:"sell_I_Value"`
		SellNValue  float64 `json:"sell_N_Value"`
	} `json:"clientType"`
}

type clientTypeResponse struct {
	ClientType []struct {
		RecDate     int64   `json:"recDate"`
		BuyICount   float64 `json:"buy_I_Count"`
		BuyIValue   float64 `json:"buy_I_Value"`
		BuyIVolume  float64 `json:"buy_I_Volume"`
		BuyNCount   float64 `json:"buy_N_Count"`
		BuyNValue   float64 `json:"buy_N_Value"`
		BuyNVolume  float64 `json:"buy_N_Volume"`
		SellICount  float64 `json:"sell_I_Count"`
		SellIValue  float64 `json:"sell_I_Value"`
		SellIVolume float64 `json:"sell_I_Volume"`
		SellNCount  float64 `json:"sell_N_Count"`
		SellNValue  float64 `json:"sell_N_Value"`
		SellNVolume float64 `json:"sell_N_Volume"`
	} `json:"clientType"`
}

func (a *Adapter) FetchInstrumentInfo(ctx context.Context, insCode string) (*InstrumentInfo, error) {
	var response instrumentInfoResponse
	if err := a.fetchJSON(ctx, "/Instrument/GetInstrumentInfo/"+url.PathEscape(insCode), &response); err != nil {
		return nil, err
	}
	if response.InstrumentInfo.InsCode == "" {
		return nil, fmt.Errorf("instrument info response has no insCode")
	}
	result := &InstrumentInfo{
		InsCode: response.InstrumentInfo.InsCode,
		Symbol:  strings.TrimSpace(response.InstrumentInfo.Symbol),
		Name:    strings.TrimSpace(response.InstrumentInfo.Name),
		Nav:     response.InstrumentInfo.Nav,
	}
	if response.InstrumentInfo.EPS != nil && response.InstrumentInfo.EPS.EstimatedEPS != nil {
		if esp, err := strconv.ParseFloat(strings.TrimSpace(*response.InstrumentInfo.EPS.EstimatedEPS), 64); err == nil {
			result.ESP = &esp
		}
	}
	return result, nil
}

func (a *Adapter) FetchInstrumentStateHistory(ctx context.Context, insCode string) ([]InstrumentStateHistory, error) {
	var response instrumentStateHistoryResponse
	path := "/MarketData/GetInstrumentStateAll/" + url.PathEscape(insCode)
	if err := a.fetchJSON(ctx, path, &response); err != nil {
		return nil, err
	}

	result := make([]InstrumentStateHistory, 0, len(response.InstrumentState))
	for _, item := range response.InstrumentState {
		date, err := parseTSEDate(item.DEven)
		if err != nil {
			return nil, fmt.Errorf("instrument state date: %w", err)
		}
		result = append(result, InstrumentStateHistory{
			ID:               item.ID,
			Date:             date,
			Time:             item.HEven,
			InsCode:          item.InsCode,
			Symbol:           trimOptionalString(item.Symbol),
			Name:             trimOptionalString(item.Name),
			Code:             strings.TrimSpace(item.Code),
			RealTime:         item.RealHEven,
			UnderSupervision: item.UnderSupervision,
			Title:            strings.TrimSpace(item.Title),
		})
	}
	return result, nil
}

func trimOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}

func (a *Adapter) FetchClosingPriceHistory(ctx context.Context, insCode string) ([]ClosingPriceHistory, error) {
	var response closingPriceResponse
	path := "/ClosingPrice/GetClosingPriceDailyList/" + url.PathEscape(insCode) + "/0"
	if err := a.fetchJSON(ctx, path, &response); err != nil {
		return nil, err
	}
	result := make([]ClosingPriceHistory, 0, len(response.ClosingPriceDaily))
	for _, item := range response.ClosingPriceDaily {
		date, err := parseTSEDate(item.DEven)
		if err != nil {
			return nil, fmt.Errorf("closing price date: %w", err)
		}
		result = append(result, ClosingPriceHistory{
			Date: date, TradeCount: int64(item.ZTotTran), Volume: int64(item.QTotTran5J), Value: int64(item.QTotCap),
			Yesterday: item.PriceYesterday, First: item.PriceFirst, Last: item.PDrCotVal,
			LastChange: item.PriceChange, Close: item.PClosing, Low: item.PriceMin, High: item.PriceMax,
		})
	}
	return result, nil
}

func (a *Adapter) FetchClientTypeHistory(ctx context.Context, insCode string) ([]ClientTypeHistory, error) {
	var response clientTypeResponse
	if err := a.fetchJSON(ctx, "/ClientType/GetClientTypeHistory/"+url.PathEscape(insCode), &response); err != nil {
		return nil, err
	}
	result := make([]ClientTypeHistory, 0, len(response.ClientType))
	for _, item := range response.ClientType {
		date, err := parseTSEDate(item.RecDate)
		if err != nil {
			return nil, fmt.Errorf("client type date: %w", err)
		}
		result = append(result, ClientTypeHistory{
			Date:                date,
			BuyIndividualVolume: int64(item.BuyIVolume), BuyIndividualValue: int64(item.BuyIValue), BuyIndividualCount: int64(item.BuyICount),
			BuyInstitutionalVolume: int64(item.BuyNVolume), BuyInstitutionalValue: int64(item.BuyNValue), BuyInstitutionalCount: int64(item.BuyNCount),
			SellIndividualVolume: int64(item.SellIVolume), SellIndividualValue: int64(item.SellIValue), SellIndividualCount: int64(item.SellICount),
			SellInstitutionalVolume: int64(item.SellNVolume), SellInstitutionalValue: int64(item.SellNValue), SellInstitutionalCount: int64(item.SellNCount),
		})
	}
	return result, nil
}

func (a *Adapter) FetchCurrentClosingPrice(ctx context.Context, insCode string) (*CurrentClosingPrice, error) {
	var response currentClosingPriceResponse
	if err := a.fetchJSON(ctx, "/ClosingPrice/GetClosingPriceInfo/"+url.PathEscape(insCode), &response); err != nil {
		return nil, fmt.Errorf("failed to fetch json price: %w", err)
	}
	item := response.ClosingPriceInfo
	result := &CurrentClosingPrice{
		InstrumentStateCode: strings.TrimSpace(item.InstrumentState.Code),
		TradeCount:          int64(item.ZTotTran),
		Volume:              int64(item.QTotTran5J),
		Value:               int64(item.QTotCap),
		Yesterday:           item.PriceYesterday,
		First:               item.PriceFirst,
		Last:                item.PDrCotVal,
		Close:               item.PClosing,
		Low:                 item.PriceMin,
		High:                item.PriceMax,
	}
	if item.DEven == 0 {
		return result, nil
	}
	date, err := parseTSEDate(item.DEven)
	if err != nil {
		return nil, fmt.Errorf("closing price date: %w", err)
	}
	result.Date = date
	result.FinalLastDate = date
	if item.FinalLastDate != 0 {
		result.FinalLastDate, err = parseTSEDate(item.FinalLastDate)
		if err != nil {
			return nil, fmt.Errorf("final closing price date: %w", err)
		}
	}
	return result, nil
}

func (a *Adapter) FetchCurrentClientType(ctx context.Context, insCode string) (*CurrentClientType, error) {
	var response currentClientTypeResponse
	path := "/ClientType/GetClientType/" + url.PathEscape(insCode) + "/1/0"
	if err := a.fetchJSON(ctx, path, &response); err != nil {
		return nil, err
	}
	item := response.ClientType
	return &CurrentClientType{
		BuyIndividualVolume:     int64(item.BuyIVolume),
		BuyIndividualCount:      int64(item.BuyICount),
		BuyIndividualValue:      int64(item.BuyIValue),
		BuyInstitutionalVolume:  int64(item.BuyNVolume),
		BuyInstitutionalCount:   int64(item.BuyNCount),
		BuyInstitutionalValue:   int64(item.BuyNValue),
		SellIndividualVolume:    int64(item.SellIVolume),
		SellIndividualCount:     int64(item.SellICount),
		SellIndividualValue:     int64(item.SellIValue),
		SellInstitutionalVolume: int64(item.SellNVolume),
		SellInstitutionalCount:  int64(item.SellNCount),
		SellInstitutionalValue:  int64(item.SellNValue),
	}, nil
}

type httpStatusError struct {
	StatusCode int
	URL        string
	Body       string
}

func (e *httpStatusError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("TSETMC returned status %d for %s", e.StatusCode, e.URL)
	}
	return fmt.Sprintf("TSETMC returned status %d for %s: %s", e.StatusCode, e.URL, e.Body)
}

func (a *Adapter) fetchJSON(ctx context.Context, path string, target any) error {
	requestURL := historyBaseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return fmt.Errorf("create TSETMC request: %w", err)
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("perform TSETMC request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return &httpStatusError{
			StatusCode: resp.StatusCode,
			URL:        requestURL,
			Body:       strings.TrimSpace(string(body)),
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode TSETMC response: %w", err)
	}
	return nil
}

func parseTSEDate(value int64) (time.Time, error) {
	date, err := time.ParseInLocation("20060102", strconv.FormatInt(value, 10), time.UTC)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid YYYYMMDD value %d: %w", value, err)
	}
	return date, nil
}
