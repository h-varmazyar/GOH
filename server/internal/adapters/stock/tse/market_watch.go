package tse

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const marketWatchURL = "https://old.tsetmc.com/tsev2/data/MarketWatchInit.aspx"

type MarketWatchRecord struct {
	InsCode        string `json:"ins_code"`         //ins_code
	Code12         string `json:"code_12"`          //isin
	Label          string `json:"label"`            //l18
	Name           string `json:"name"`             //l30
	LastDateUpdate string `json:"last_date_update"` //heven
	First          string `json:"first"`            //pf
	Close          string `json:"close"`            //pc
	Last           string `json:"last"`             //pl
	TradeCount     string `json:"trade_count"`      //tno
	TradeVolume    string `json:"trade_volume"`     //tvol
	TradeValue     string `json:"trade_value"`      //tval
	MinPrice       string `json:"min_price"`        //pmin
	MaxPrice       string `json:"max_price"`        //pmax
	Yesterday      string `json:"yesterday"`        //py
	EPS            string `json:"eps"`              //eps
	BVol           string `json:"b_vol"`            //bvol
	VisitCount     string `json:"visit_count"`      //visitcount
	Flow           string `json:"flow"`             //flow
	CS             string `json:"cs"`               //cs
	LegMax         string `json:"leg_max"`          //tmax
	LegMin         string `json:"leg_min"`          //tmin
	UnitCount      string `json:"unit_count"`       //z
	YesterdayValue string `json:"yesterday_value"`  //yval
	U23            string `json:"u23"`              //predtran
	OpenPositions  string `json:"open_positions"`   //buyop
	U25            string `json:"u25"`              //cgrvalcot
}

// GetMarketWatch fetches all TSETMC market records and publishes each one
// using the same response shape as the crypto candle pipeline.
func (n *Adapter) GetMarketWatch(ctx context.Context, heven, ref int64) ([]MarketWatchRecord, int64, error) {
	marketURL := fmt.Sprintf("%s?h=%d&r=%d", marketWatchURL, heven, ref)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, marketURL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("create TSE market watch request: %w", err)
	}

	resp, err := n.httlClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request TSE market watch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("TSE market watch returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("read TSE market watch response: %w", err)
	}

	records, refID, err := parseMarketWatch(body)
	if err != nil {
		return nil, 0, err
	}

	return records, refID, nil
}

func parseMarketWatch(data []byte) ([]MarketWatchRecord, int64, error) {
	parts := strings.Split(string(data), "@")
	if len(parts) < 4 {
		return nil, 0, fmt.Errorf("invalid TSE market watch response: expected at least 3 sections")
	}

	rows := strings.Split(parts[2], ";")
	records := make([]MarketWatchRecord, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row) == "" {
			continue
		}

		fields := strings.Split(row, ",")
		if len(fields) < 26 {
			continue
		}

		records = append(records, MarketWatchRecord{
			InsCode:        fields[0],
			Code12:         fields[1],
			Label:          fields[2],
			Name:           fields[3],
			LastDateUpdate: fields[4],
			First:          fields[5],
			Close:          fields[6],
			Last:           fields[7],
			TradeCount:     fields[8],
			TradeVolume:    fields[9],
			TradeValue:     fields[10],
			MinPrice:       fields[11],
			MaxPrice:       fields[12],
			Yesterday:      fields[13],
			EPS:            fields[14],
			BVol:           fields[15],
			VisitCount:     fields[16],
			Flow:           fields[17],
			CS:             fields[18],
			LegMax:         fields[19],
			LegMin:         fields[20],
			UnitCount:      fields[21],
			YesterdayValue: fields[22],
			U23:            fields[23],
			OpenPositions:  fields[24],
			U25:            fields[25],
		})
	}

	return records, prepareRefID(parts[4]), nil
}

func prepareRefID(ref string) int64 {
	refID, err := strconv.Atoi(ref)
	if err != nil {
		return 0
	}

	return int64(25 * (refID / 25))
}
