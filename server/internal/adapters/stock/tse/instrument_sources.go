package tse

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/sirupsen/logrus"
)

const (
	tseCashMarketListingURL = "https://webgw.tse.ir/cmssite/api/v1/EDataListingCashMarket/List/fa"
	tseETFListURL           = "https://webgw.tse.ir/InstrumentProvider/api/v1/ArchiveMarket/ETFMarket"

	instrumentsAssetsDir = "./assets/tse/instruments"
)

// localInstrument is an instrument of the TSETMC instrument files.
type localInstrument struct {
	InsCode      string
	Symbol       string
	Name         string
	Group        string
	SubGroup     string
	Type         int
	MarketCode   int
	SymbolStatus string
	Active       bool
}

// sourceListing is a TSE listing (cash market or ETF) keyed by instrument ID.
type sourceListing struct {
	InstrumentID string
	Symbol       string
	Name         string
	InsertDate   *time.Time
	ReleaseDate  *time.Time
}

// ifbBoardListing is a symbol of the IFB first or second market with its dates.
type ifbBoardListing struct {
	ifbMarketRow
	MarketType     consts.TSEMarketType
	InsertDate     *time.Time
	ReleaseDate    *time.Time
	AcceptanceDate *time.Time
}

type catalogSources struct {
	byID      map[string]sourceListing
	ifbBoards []ifbBoardListing
	groups    map[string]string
}

type instrumentCompareResponse struct {
	Items []struct {
		Instruments []struct {
			InsCode      string `json:"insCode"`
			Symbol       string `json:"smallSymbolName"`
			Name         string `json:"symbolFullName"`
			Group        string `json:"industryGroupCode"`
			SubGroup     string `json:"industrySubCode"`
			SymbolStatus string `json:"symbolStatus"`
			Type         int    `json:"type"`
			MarketCode   int    `json:"marketCode"`
			Status       int    `json:"status"`
		} `json:"instruments"`
	} `json:"items"`
}

type tseCashListingItem struct {
	InstrumentID string  `json:"instrumentId"`
	Symbol       string  `json:"instrumentName"`
	CompanyName  string  `json:"companyName"`
	InsertDate   *string `json:"insertDate"`
	ReleaseDate  *string `json:"releaseDate"`
}

type tseETFItem struct {
	InstrumentID          string  `json:"instrumentId"`
	Symbol                string  `json:"instrumentname"`
	Name                  string  `json:"persian30insname"`
	SubscriptionStart     *string `json:"tarixShoruPazireNevisi"`
	SecondaryTradingStart *string `json:"tarixShoruMoamelatSanavie"`
}

type staticDataResponse struct {
	StaticData []struct {
		Code int    `json:"code"`
		Type string `json:"type"`
		Name string `json:"name"`
	} `json:"staticData"`
}

func loadLocalInstruments(ctx context.Context, dir string) ([]localInstrument, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read TSE instruments directory: %w", err)
	}
	result := make([]localInstrument, 0)
	seen := make(map[string]bool)
	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".json" {
			continue
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		path := filepath.Join(dir, entry.Name())
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
				insCode := strings.TrimSpace(instrument.InsCode)
				if insCode == "" || seen[insCode] {
					continue
				}
				seen[insCode] = true
				result = append(result, localInstrument{
					InsCode:      insCode,
					Symbol:       strings.TrimSpace(instrument.Symbol),
					Name:         strings.TrimSpace(instrument.Name),
					Group:        strings.TrimSpace(instrument.Group),
					SubGroup:     strings.TrimSpace(instrument.SubGroup),
					Type:         instrument.Type,
					MarketCode:   instrument.MarketCode,
					SymbolStatus: strings.TrimSpace(instrument.SymbolStatus),
					Active:       instrument.Status == 1,
				})
			}
		}
	}
	return result, nil
}

func (a *Adapter) fetchCatalogSources(ctx context.Context) (*catalogSources, error) {
	logger := catalogLogger(a)
	sources := &catalogSources{byID: make(map[string]sourceListing)}
	if err := a.fetchTSECashListings(ctx, sources.byID); err != nil {
		return nil, fmt.Errorf("fetch TSE cash market listings: %w", err)
	}
	cash := len(sources.byID)
	logger.WithField("listings", cash).Info("fetched TSE cash market listings")
	if err := a.fetchTSEETFListings(ctx, sources.byID); err != nil {
		return nil, fmt.Errorf("fetch TSE ETF listings: %w", err)
	}
	logger.WithFields(logrus.Fields{"new_listings": len(sources.byID) - cash, "listings": len(sources.byID)}).
		Info("fetched TSE ETF listings")
	boards, err := a.fetchIFBBoards(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch IFB listings: %w", err)
	}
	sources.ifbBoards = boards
	perBoard := make(logrus.Fields)
	for _, board := range boards {
		count, _ := perBoard[string(board.MarketType)].(int)
		perBoard[string(board.MarketType)] = count + 1
	}
	logger.WithFields(perBoard).WithField("symbols", len(boards)).Info("fetched IFB first and second market symbols")
	if sources.groups, err = a.fetchIndustryGroups(ctx); err != nil {
		logger.WithError(err).Warn("fetch TSE industry groups; group names fall back to TSETMC sectors")
	} else {
		logger.WithField("groups", len(sources.groups)).Info("fetched TSE industry groups")
	}
	return sources, nil
}

// fetchTSECashListings reads the TSE cash market listings. Besides TSE stocks the
// list has IFB stocks and funds; every entry is kept and joined by instrument ID.
func (a *Adapter) fetchTSECashListings(ctx context.Context, byID map[string]sourceListing) error {
	var items []tseCashListingItem
	if err := a.getJSON(ctx, tseCashMarketListingURL, &items); err != nil {
		return err
	}
	for _, item := range items {
		id := strings.TrimSpace(item.InstrumentID)
		if !strings.HasPrefix(id, "IR") {
			continue
		}
		if _, exists := byID[id]; exists {
			continue
		}
		byID[id] = sourceListing{
			InstrumentID: id,
			Symbol:       cleanText(item.Symbol),
			Name:         cleanText(item.CompanyName),
			InsertDate:   optionalISODate(item.InsertDate),
			ReleaseDate:  optionalISODate(item.ReleaseDate),
		}
	}
	return nil
}

// fetchTSEETFListings maps the subscription (پذیره‌نویسی) start date to the
// release date; funds without a subscription period use their first trading day.
func (a *Adapter) fetchTSEETFListings(ctx context.Context, byID map[string]sourceListing) error {
	var items []tseETFItem
	if err := a.getJSON(ctx, tseETFListURL, &items); err != nil {
		return err
	}
	for _, item := range items {
		id := strings.TrimSpace(item.InstrumentID)
		if !strings.HasPrefix(id, "IR") {
			continue
		}
		release := optionalISODate(item.SubscriptionStart)
		if release == nil {
			release = optionalISODate(item.SecondaryTradingStart)
		}
		listing := byID[id]
		listing.InstrumentID = id
		listing.Symbol = cleanText(item.Symbol)
		listing.Name = cleanText(item.Name)
		if listing.ReleaseDate == nil {
			listing.ReleaseDate = release
		}
		byID[id] = listing
	}
	return nil
}

// fetchIFBBoards takes the symbols of the IFB first and second markets and
// completes them with the dates of the IFB listing page.
func (a *Adapter) fetchIFBBoards(ctx context.Context) ([]ifbBoardListing, error) {
	dates, err := a.fetchIFBListings(ctx)
	if err != nil {
		return nil, err
	}
	markets := []struct {
		id         int
		marketType consts.TSEMarketType
	}{
		{ifbFirstMarketID, consts.TSEMarketTypeIFBFirst},
		{ifbSecondMarketID, consts.TSEMarketTypeIFBSecond},
	}
	result := make([]ifbBoardListing, 0)
	for _, market := range markets {
		rows, err := a.fetchIFBMarket(ctx, market.id)
		if err != nil {
			return nil, fmt.Errorf("market %d: %w", market.id, err)
		}
		for _, row := range rows {
			item := ifbBoardListing{ifbMarketRow: row, MarketType: market.marketType}
			if date, ok := dates[row.SymID]; ok {
				item.InsertDate = date.InsertDate
				item.ReleaseDate = date.ReleaseDate
				item.AcceptanceDate = date.AcceptanceDate
				if item.Name == "" {
					item.Name = date.Name
				}
			}
			result = append(result, item)
		}
	}
	return result, nil
}

// fetchIndustryGroups returns TSETMC industry group names keyed by groupKey.
func (a *Adapter) fetchIndustryGroups(ctx context.Context) (map[string]string, error) {
	var r staticDataResponse
	if err := a.fetchJSON(ctx, fetchStaticDataURL, &r); err != nil {
		return nil, err
	}
	result := make(map[string]string)
	for _, item := range r.StaticData {
		if item.Type == "IndustrialGroup" {
			result[strconv.Itoa(item.Code)] = cleanText(item.Name)
		}
	}
	return result, nil
}

// groupKey drops the padding and leading zeros of TSETMC group codes ("01 " -> "1").
func groupKey(code string) string {
	code = strings.TrimSpace(code)
	if value, err := strconv.Atoi(code); err == nil {
		return strconv.Itoa(value)
	}
	return code
}

func optionalISODate(value *string) *time.Time {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if len(trimmed) < len(time.DateOnly) {
		return nil
	}
	date, err := time.ParseInLocation(time.DateOnly, trimmed[:len(time.DateOnly)], time.UTC)
	if err != nil {
		return nil
	}
	return &date
}
