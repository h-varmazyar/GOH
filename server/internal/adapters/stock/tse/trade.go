package tse

import (
	"context"
	"fmt"
	"net/url"
)

type tradeHistoryResponse struct {
	TradeHistory []TradeHistory `json:"tradeHistory"`
}

type TradeHistory struct {
	Canceled  int     `json:"canceled"`
	TradeTime int     `json:"hEven"`
	Price     float64 `json:"pTran"`
	Quantity  int     `json:"qTitTran"`
}

func (a *Adapter) FetchTradeHistory(ctx context.Context, insCode, date string) ([]TradeHistory, error) {
	var r tradeHistoryResponse
	if err := a.fetchJSON(ctx, fmt.Sprintf(fetchTradeHistoryURL, url.PathEscape(insCode), url.PathEscape(date)), &r); err != nil {
		return nil, err
	}
	return r.TradeHistory, nil
}
