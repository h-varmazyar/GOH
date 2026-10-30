package tse

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

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

func (a *Adapter) FetchCurrentClientType(ctx context.Context, insCode string) (*ClientType, error) {
	var r currentClientResponse
	if err := a.fetchJSON(ctx, fmt.Sprintf(fetchCurrentClientTypeURL, url.PathEscape(insCode)), &r); err != nil {
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

func (a *Adapter) FetchClientTypeHistory(ctx context.Context, insCode string) ([]ClientType, error) {
	var r clientHistoryResponse
	if err := a.fetchJSON(ctx, fmt.Sprintf(fetchClientTypeHistory, url.PathEscape(insCode)), &r); err != nil {
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
