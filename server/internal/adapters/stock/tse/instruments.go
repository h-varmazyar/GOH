package tse

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// InstrumentInfo is the TSETMC instrument info. Identification and classification
// fields rarely change; Nav, EPS, SectorPE, PSR, TotalShares and BaseVolume do.
type InstrumentInfo struct {
	InsCode      string
	InstrumentID string
	CompanyISIN  string
	Symbol       string
	Name         string
	EnglishName  string
	Type         int
	Flow         int
	BoardCode    string
	SectorCode   string
	SectorName   string
	TotalShares  *float64
	BaseVolume   *float64
	Nav          *float64
	EPS          *float64
	SectorPE     *float64
	PSR          *float64
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
		InsCode      string   `json:"insCode"`
		InstrumentID string   `json:"instrumentID"`
		CompanyISIN  *string  `json:"cIsin"`
		Symbol       string   `json:"lVal18AFC"`
		Name         string   `json:"lVal30"`
		EnglishName  string   `json:"lVal18"`
		Type         string   `json:"yVal"`
		Flow         int      `json:"flow"`
		BoardCode    string   `json:"cgrValCot"`
		TotalShares  *float64 `json:"zTitad"`
		BaseVolume   *float64 `json:"baseVol"`
		Nav          *float64 `json:"nav"`
		Sector       *struct {
			Code string `json:"cSecVal"`
			Name string `json:"lSecVal"`
		} `json:"sector"`
		EPS *struct {
			EstimatedEPS *string  `json:"estimatedEPS"`
			SectorPE     *float64 `json:"sectorPE"`
			PSR          *float64 `json:"psr"`
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

// UpdateInstruments returns every stock, fund and right of the Tehran Stock
// Exchange and Iran Fara Bourse found in the TSETMC instrument files, including
// delisted ones, plus listed instruments that are missing from the files.
//
// Problems that need a decision of the user are returned in Catalog.Problems.
// The TSETMC instrument info of every instrument is fetched to verify its type,
// symbol and instrument ID (IR...), and is returned in Instrument.Info. Listing
// dates and names are joined by instrument ID; IFB pages carry no instrument ID,
// so their board is matched by symbol, type and name.
func (n *Adapter) UpdateInstruments(ctx context.Context) (*Catalog, error) {
	logger := catalogLogger(n)
	start := time.Now()
	local, err := loadLocalInstruments(ctx, instrumentsAssetsDir)
	if err != nil {
		return nil, err
	}
	logger.WithFields(logrus.Fields{"dir": instrumentsAssetsDir, "instruments": len(local)}).Info("loaded TSE instrument files")
	logger.Info("fetching TSE and IFB listing sources")
	sources, err := n.fetchCatalogSources(ctx)
	if err != nil {
		return nil, err
	}
	catalog, err := newCatalogBuilder(n, sources).build(ctx, local)
	if err != nil {
		return nil, err
	}
	logger.WithFields(logrus.Fields{
		"instruments": len(catalog.Instruments), "problems": len(catalog.Problems), "duration": time.Since(start).Round(time.Second),
	}).Info("built TSE instrument catalog")
	return catalog, nil
}

func catalogLogger(adapter *Adapter) *logrus.Entry {
	return logrus.WithFields(logrus.Fields{"component": "tse-catalog", "provider": adapter.GetID()})
}

// ResolveInstrument builds the instrument of a single insCode, for symbols that are
// added or refreshed by hand. A non-empty symbol must equal the TSETMC symbol of the
// insCode. The instrument files provide the delisted state and the subgroup of
// known insCodes; other insCodes are taken as active.
func (n *Adapter) ResolveInstrument(ctx context.Context, insCode, symbol string) (Instrument, error) {
	insCode = strings.TrimSpace(insCode)
	info, err := n.FetchInstrumentInfo(ctx, insCode)
	if err != nil {
		return Instrument{}, fmt.Errorf("fetch TSETMC instrument info: %w", err)
	}
	if symbol != "" && matchKey(info.Symbol) != matchKey(symbol) {
		return Instrument{}, fmt.Errorf("symbol %q does not match TSETMC symbol %q of insCode %s", symbol, cleanText(info.Symbol), insCode)
	}
	if _, ok := catalogTypes[info.Type]; !ok {
		return Instrument{}, fmt.Errorf("TSETMC instrument type %d of insCode %s is not a stock, fund or right", info.Type, insCode)
	}
	local, err := loadLocalInstruments(ctx, instrumentsAssetsDir)
	if err != nil {
		return Instrument{}, err
	}
	entry := &catalogEntry{localInstrument: localInstrument{
		InsCode:      insCode,
		Symbol:       cleanText(info.Symbol),
		Name:         info.Name,
		Group:        info.SectorCode,
		Type:         info.Type,
		SymbolStatus: "A",
		Active:       true,
	}}
	for _, item := range local {
		if item.InsCode == insCode {
			entry.localInstrument = item
			break
		}
	}
	if err := entry.applyInfo(info); err != nil {
		return Instrument{}, fmt.Errorf("insCode %s: %w", insCode, err)
	}
	sources, err := n.fetchCatalogSources(ctx)
	if err != nil {
		return Instrument{}, err
	}
	// Problems of the IFB board rows of other symbols are not about this insCode.
	instrument := newCatalogBuilder(n, sources).classify([]*catalogEntry{entry})[0]
	catalogLogger(n).WithFields(logrus.Fields{
		"ins_code": instrument.InsCode, "symbol": instrument.Symbol, "instrument_id": instrument.InstrumentID,
		"market_type": instrument.MarketType, "kind": instrument.Kind, "delisted": instrument.Delisted,
	}).Info("resolved TSE instrument")
	return instrument, nil
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
	item := r.InstrumentInfo
	if item.InsCode == "" {
		return nil, fmt.Errorf("instrument info response has no insCode")
	}
	result := &InstrumentInfo{
		InsCode:      strings.TrimSpace(item.InsCode),
		InstrumentID: strings.TrimSpace(item.InstrumentID),
		Symbol:       strings.TrimSpace(item.Symbol),
		Name:         strings.TrimSpace(item.Name),
		EnglishName:  strings.TrimSpace(item.EnglishName),
		Flow:         item.Flow,
		BoardCode:    strings.TrimSpace(item.BoardCode),
		TotalShares:  item.TotalShares,
		BaseVolume:   item.BaseVolume,
		Nav:          item.Nav,
	}
	if item.CompanyISIN != nil {
		result.CompanyISIN = strings.TrimSpace(*item.CompanyISIN)
	}
	if value, err := strconv.Atoi(strings.TrimSpace(item.Type)); err == nil {
		result.Type = value
	}
	if item.Sector != nil {
		result.SectorCode = strings.TrimSpace(item.Sector.Code)
		result.SectorName = cleanText(item.Sector.Name)
	}
	if item.EPS != nil {
		if item.EPS.EstimatedEPS != nil {
			if value, err := strconv.ParseFloat(strings.TrimSpace(*item.EPS.EstimatedEPS), 64); err == nil {
				result.EPS = &value
			}
		}
		result.SectorPE = item.EPS.SectorPE
		result.PSR = item.EPS.PSR
	}
	return result, nil
}
