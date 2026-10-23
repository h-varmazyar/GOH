package nobitex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/h-varmazyar/gatec/internal/adapters/crypto"
	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/sirupsen/logrus"
)

type marketList struct {
	baseResponse
	Stats map[string]struct {
		IsClosed bool `json:"isClosed"`
	}
}

func (n *Adapter) UpdateMarkets(ctx context.Context) ([]crypto.Market, error) {
	resp, err := n.httlClient.Get(fmt.Sprintf("%v/market/stats", n.cfg.APIBaseURL))
	if err != nil {
		logrus.WithError(err).Error("failed to make market update request in nobitex adapter")
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logrus.WithError(err).Errorf("failed to read market list body")
		return nil, err
	}
	defer resp.Body.Close()

	marketsStats := new(marketList)
	if err = json.Unmarshal(body, marketsStats); err != nil {
		logrus.WithError(err).Error("failed to unmarshal body for update market in nobitex adapter")
		return nil, err
	}

	if marketsStats.Status != "ok" {
		return nil, errors.New(marketsStats.Message)
	}

	markets := make([]crypto.Market, 0)
	for marketName, market := range marketsStats.Stats {
		name := strings.ToUpper(marketName)
		status := consts.MarketStatusOpen
		if market.IsClosed {
			status = consts.MarketStatusClose
		}
		m := crypto.Market{
			Name:   strings.Replace(name, "RLS", "IRT", -1),
			Status: status,
		}

		markets = append(markets, m)
	}

	return markets, nil
}
