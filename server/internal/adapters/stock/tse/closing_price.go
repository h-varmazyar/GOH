package tse

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

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

func (a *Adapter) FetchCurrentClosingPrice(ctx context.Context, insCode string) (*ClosingPrice, error) {
	var r currentClosingResponse
	if err := a.fetchJSON(ctx, fmt.Sprintf(fetchCurrentClosingPriceURL, url.PathEscape(insCode)), &r); err != nil {
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

func (a *Adapter) FetchClosingPriceHistory(ctx context.Context, insCode string) ([]ClosingPriceHistory, error) {
	var r closingPriceResponse
	if err := a.fetchJSON(ctx, fmt.Sprintf(fetchClosingPriceHistoryURL, url.PathEscape(insCode)), &r); err != nil {
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
