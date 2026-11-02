package tsemarket

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/h-varmazyar/gatec/internal/adapters/stock/tse"
	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/h-varmazyar/gatec/pkg/paginator"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	pricingAssetLabel = "IRI"
	// Instruments whose name is less similar than this to the name of their symbol's
	// market are reported, so that their link can be checked in the panel.
	minMemberNameSimilarity = 0.5
	// syncProgressEvery is how often the progress of storing the catalog is logged.
	syncProgressEvery = 500
)

var (
	ErrNotFound      = errors.New("TSE market not found")
	ErrAlreadyExists = errors.New("TSE market already exists")
	ErrNoAdapter     = errors.New("no active TSE provider")
	ErrInvalidInput  = errors.New("invalid input")
)

// Service keeps the TSE markets in line with the TSE instrument catalog and
// serves the panel.
//
// Each insCode is a TSE market with its own prices. A symbol gets a new insCode
// whenever it moves between boards, so the insCodes of a symbol share one market
// that holds the history of the symbol.
type Service struct {
	tseRepo    *repositories.TSEMarketRepository
	marketRepo *repositories.MarketRepository
	assetRepo  *repositories.AssetRepository
	adapter    *tse.Adapter

	// detectMu serializes issue checks; detectState guards the background check.
	detectMu      sync.Mutex
	detectState   sync.Mutex
	detectRunning bool
	detectAgain   bool
}

func NewService(
	tseRepo *repositories.TSEMarketRepository,
	marketRepo *repositories.MarketRepository,
	assetRepo *repositories.AssetRepository,
	adapter *tse.Adapter,
) *Service {
	return &Service{tseRepo: tseRepo, marketRepo: marketRepo, assetRepo: assetRepo, adapter: adapter}
}

func (s *Service) logger() *logrus.Entry {
	logger := logrus.WithField("service", "tsemarket")
	if s.adapter != nil {
		logger = logger.WithField("provider", s.adapter.GetID())
	}
	return logger
}

// syncState holds the stored markets of the provider while instruments are stored.
type syncState struct {
	providerID  uint
	destination *models.Asset
	byInsCode   map[string]*models.TSEMarket
	// markets are the markets of the provider keyed by tse.SymbolKey of their label.
	markets map[string]*models.Market
	// kinds is the instrument kind of the members of each market.
	kinds map[uint]consts.TSEInstrumentKind
	// named are the markets whose name and label were set while the state was used.
	named map[uint]bool
	// problems are found while instruments are stored.
	problems []tse.CatalogProblem
	// createdMarkets counts the markets created while the state was used.
	createdMarkets int
}

func (s *Service) loadSyncState(ctx context.Context) (*syncState, error) {
	if s.adapter == nil {
		return nil, ErrNoAdapter
	}
	destination, err := s.ensureAsset(ctx, pricingAssetLabel, "ریال ایران")
	if err != nil {
		return nil, fmt.Errorf("ensure pricing asset: %w", err)
	}
	markets, err := s.marketRepo.List(ctx, map[string]any{"provider_id = ?": s.adapter.GetID()})
	if err != nil {
		return nil, fmt.Errorf("list markets: %w", err)
	}
	tseMarkets, err := s.tseRepo.List(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("list TSE markets: %w", err)
	}
	state := &syncState{
		providerID:  s.adapter.GetID(),
		destination: destination,
		byInsCode:   make(map[string]*models.TSEMarket, len(tseMarkets)),
		markets:     make(map[string]*models.Market, len(markets)),
		kinds:       make(map[uint]consts.TSEInstrumentKind, len(markets)),
		named:       make(map[uint]bool),
	}
	byID := make(map[uint]*models.Market, len(markets))
	for _, market := range markets {
		state.markets[tse.SymbolKey(market.Label)] = market
		byID[market.ID] = market
	}
	for _, market := range tseMarkets {
		// Members of a market share one market value, so its updates reach all of them.
		if shared := byID[market.MarketID]; shared != nil {
			market.Market = shared
		}
		state.byInsCode[market.InsCode] = market
		state.kinds[market.MarketID] = market.Kind
	}
	return state, nil
}

// SyncResult summarizes a catalog sync.
type SyncResult struct {
	Instruments int
	Created     int
	Updated     int
	Failed      int
	// Markets is the number of markets created for new symbols.
	Markets int
	// Issues is the number of issues found in the stored markets.
	Issues int
}

// SyncCatalog stores the TSE instrument catalog. New instruments are added and
// linked to the market of their symbol; stored instruments are verified against
// the sources and TSETMC. Panel flags, market links and delisted marks set in the
// panel are kept. Problems that need a decision are stored as issues for the
// panel, and the stored markets are checked for issues afterwards.
func (s *Service) SyncCatalog(ctx context.Context) (SyncResult, error) {
	var result SyncResult
	logger := s.logger()
	start := time.Now()
	state, err := s.loadSyncState(ctx)
	if err != nil {
		return result, err
	}
	stored := len(state.byInsCode)
	logger.WithFields(logrus.Fields{"tse_markets": stored, "markets": len(state.markets)}).Info("loaded stored TSE markets")

	logger.Info("building TSE instrument catalog")
	catalog, err := s.adapter.UpdateInstruments(ctx)
	if err != nil {
		return result, fmt.Errorf("fetch instruments: %w", err)
	}
	instruments := catalog.Instruments
	result.Instruments = len(instruments)

	logger.WithField("instruments", len(instruments)).Info("storing TSE instruments")
	storeStart := time.Now()
	progress := func(done int) logrus.Fields {
		created := len(state.byInsCode) - stored
		return logrus.Fields{
			"done": done, "total": len(instruments), "created": created, "updated": done - created - result.Failed,
			"failed": result.Failed, "markets_created": state.createdMarkets, "elapsed": time.Since(storeStart).Round(time.Second),
		}
	}
	// Instruments come best first, so the active instrument of a symbol names its market.
	for i, instrument := range instruments {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if _, err := s.store(ctx, state, instrument, false); err != nil {
			result.Failed++
			logger.WithError(err).WithFields(logrus.Fields{"ins_code": instrument.InsCode, "symbol": instrument.Symbol}).
				Error("failed to store TSE instrument")
		}
		if (i+1)%syncProgressEvery == 0 {
			logger.WithFields(progress(i + 1)).Info("storing TSE instruments")
		}
	}
	result.Created = len(state.byInsCode) - stored
	result.Updated = len(instruments) - result.Created - result.Failed
	result.Markets = state.createdMarkets
	logger.WithFields(progress(len(instruments))).WithField("store_problems", len(state.problems)).Info("stored TSE instruments")

	if err := s.saveProblems(ctx, state, append(catalog.Problems, state.problems...), start); err != nil {
		return result, err
	}
	detection, err := s.DetectIssues(ctx)
	result.Issues = detection.Found
	if err == nil {
		logger.WithFields(logrus.Fields{
			"instruments": result.Instruments, "created": result.Created, "updated": result.Updated, "failed": result.Failed,
			"markets_created": result.Markets, "issues": result.Issues, "duration": time.Since(start).Round(time.Second),
		}).Info("synced TSE instrument catalog")
	}
	return result, err
}

// store stores the catalog fields and the TSETMC info of an instrument.
func (s *Service) store(ctx context.Context, state *syncState, instrument tse.Instrument, manual bool) (*models.TSEMarket, error) {
	market, err := s.upsert(ctx, state, instrument, manual)
	if err != nil {
		return nil, err
	}
	if err := s.SaveInfo(ctx, market, instrument.Info); err != nil {
		return nil, err
	}
	return market, nil
}

// upsert stores the catalog fields of an instrument. Panel flags and the market
// link of stored instruments are never changed, and a delisted mark is only
// cleared in the panel. Trading status is not tracked here.
func (s *Service) upsert(ctx context.Context, state *syncState, instrument tse.Instrument, manual bool) (*models.TSEMarket, error) {
	if instrument.InsCode == "" || instrument.Symbol == "" {
		return nil, fmt.Errorf("%w: instrument has no insCode or symbol", ErrInvalidInput)
	}
	if existing := state.byInsCode[instrument.InsCode]; existing != nil {
		if err := s.tseRepo.Update(ctx, existing.ID, catalogColumns(instrument)); err != nil {
			return nil, fmt.Errorf("update TSE market %s: %w", instrument.InsCode, err)
		}
		delisted := existing.Delisted || instrument.Delisted
		applyCatalogFields(existing, instrument)
		existing.Delisted = delisted
		if err := s.updateMarket(ctx, state, existing, instrument); err != nil {
			return nil, err
		}
		return existing, nil
	}

	market, err := s.marketFor(ctx, state, instrument)
	if err != nil {
		return nil, err
	}
	tseMarket := &models.TSEMarket{
		InsCode:               instrument.InsCode,
		ManuallyAdded:         manual,
		DataCollectionEnabled: true,
	}
	applyCatalogFields(tseMarket, instrument)
	newMarket := market.ID == 0
	if newMarket {
		err = s.tseRepo.CreateWithMarket(ctx, market, tseMarket)
	} else {
		tseMarket.MarketID = market.ID
		err = s.tseRepo.Create(ctx, tseMarket)
	}
	if err != nil {
		return nil, fmt.Errorf("create TSE market %s of market %q: %w", instrument.InsCode, market.Label, err)
	}
	tseMarket.Market = market
	state.byInsCode[tseMarket.InsCode] = tseMarket
	state.markets[tse.SymbolKey(market.Label)] = market
	state.kinds[market.ID] = tseMarket.Kind
	if newMarket {
		state.named[market.ID] = true
		state.createdMarkets++
		return tseMarket, nil
	}
	return tseMarket, s.updateMarket(ctx, state, tseMarket, instrument)
}

// marketFor returns the market of the symbol of a new instrument, or a new market
// when the symbol has none. An instrument of another kind than the members of the
// symbol's market gets a separate market labelled with its insCode.
func (s *Service) marketFor(ctx context.Context, state *syncState, instrument tse.Instrument) (*models.Market, error) {
	label := instrument.Symbol
	if market := state.markets[tse.SymbolKey(label)]; market != nil {
		kind := state.kinds[market.ID]
		if kind == "" || kind == instrument.Kind {
			if similarity := tse.NameSimilarity(market.Name, instrument.Name); market.Name != "" && similarity < minMemberNameSimilarity {
				s.logger().WithFields(logrus.Fields{
					"symbol": instrument.Symbol, "ins_code": instrument.InsCode, "name": instrument.Name,
					"market": market.Label, "market_name": market.Name, "similarity": similarity,
				}).Warn("TSE instrument name differs from the market of its symbol; check its market in the panel")
			}
			return market, nil
		}
		label = fmt.Sprintf("%s-%s", instrument.Symbol, instrument.InsCode)
		state.problems = append(state.problems, tse.CatalogProblem{
			Type: consts.TSEIssueKindConflict, InsCode: instrument.InsCode, Symbol: instrument.Symbol, Name: instrument.Name,
			Detail: fmt.Sprintf("نماد «%s» در مارکتی از نوع %s است ولی این insCode از نوع %s است؛ مارکت جداگانه «%s» برایش ساخته شد.",
				instrument.Symbol, kind.PersianLabel(), instrument.Kind.PersianLabel(), label),
		})
	}
	trading, err := s.ensureAsset(ctx, instrument.Symbol, instrument.Name)
	if err != nil {
		return nil, fmt.Errorf("ensure trading asset: %w", err)
	}
	return &models.Market{
		ProviderID:     state.providerID,
		PricingAssetID: state.destination.ID,
		TradingAssetID: trading.ID,
		Name:           instrument.Name,
		Label:          label,
		// Initial value only; the trading state is maintained by a separate worker.
		Status: consts.MarketStatusOpen,
	}, nil
}

// updateMarket keeps the name and label of a market in line with its first active
// instrument handled by the state. Delisted instruments never rename a market.
func (s *Service) updateMarket(ctx context.Context, state *syncState, tseMarket *models.TSEMarket, instrument tse.Instrument) error {
	market := tseMarket.Market
	if market == nil || tseMarket.Delisted || state.named[market.ID] {
		return nil
	}
	state.named[market.ID] = true
	columns := map[string]any{}
	if instrument.Name != "" && market.Name != instrument.Name {
		columns["name"] = instrument.Name
	}
	labelKey, symbolKey := tse.SymbolKey(market.Label), tse.SymbolKey(instrument.Symbol)
	switch owner := state.markets[symbolKey]; {
	case labelKey == symbolKey:
	case owner == nil:
		columns["label"] = instrument.Symbol
	case !strings.HasPrefix(labelKey, symbolKey):
		state.problems = append(state.problems, tse.CatalogProblem{
			Type: consts.TSEIssueLabelConflict, InsCode: instrument.InsCode, Symbol: instrument.Symbol, Name: instrument.Name,
			Detail: fmt.Sprintf("نماد این insCode به «%s» تغییر کرده که برچسب مارکت دیگری است؛ برچسب مارکت «%s» تغییر نکرد.",
				instrument.Symbol, market.Label),
		})
	}
	if len(columns) == 0 {
		return nil
	}
	if err := s.marketRepo.Update(ctx, market.ID, columns); err != nil {
		return fmt.Errorf("update market %d: %w", market.ID, err)
	}
	if label, ok := columns["label"].(string); ok {
		delete(state.markets, labelKey)
		state.markets[symbolKey] = market
		market.Label = label
	}
	if name, ok := columns["name"].(string); ok {
		market.Name = name
	}
	return nil
}

// catalogColumns are the stored columns that the catalog owns. Missing dates do
// not erase stored ones; a release date found later in price history is kept
// until a listing source provides one. Delisted is only ever set.
func catalogColumns(instrument tse.Instrument) map[string]any {
	columns := map[string]any{
		"label":           instrument.Symbol,
		"name":            instrument.Name,
		"market_type":     instrument.MarketType,
		"kind":            instrument.Kind,
		"instrument_type": instrument.InstrumentType,
	}
	if instrument.Delisted {
		columns["delisted"] = true
	}
	if instrument.InstrumentID != "" {
		columns["instrument_id"] = instrument.InstrumentID
	}
	if instrument.Group != "" {
		columns["industry_group_code"] = instrument.Group
	}
	if instrument.GroupName != "" {
		columns["industry_group_name"] = instrument.GroupName
	}
	if instrument.SubGroup != "" {
		columns["industry_sub_code"] = instrument.SubGroup
	}
	if instrument.InsertDate != nil {
		columns["insert_date"] = *instrument.InsertDate
	}
	if instrument.AcceptanceDate != nil {
		columns["acceptance_date"] = *instrument.AcceptanceDate
	}
	if instrument.ReleaseDate != nil {
		columns["ipo_date"] = *instrument.ReleaseDate
		columns["ipo_date_source"] = models.TSEIPODateSourceListing
	}
	return columns
}

func applyCatalogFields(market *models.TSEMarket, instrument tse.Instrument) {
	market.Label = instrument.Symbol
	market.Name = instrument.Name
	market.MarketType = instrument.MarketType
	market.Kind = instrument.Kind
	market.InstrumentType = instrument.InstrumentType
	market.Delisted = instrument.Delisted
	if instrument.InstrumentID != "" {
		market.InstrumentID = instrument.InstrumentID
	}
	if instrument.Group != "" {
		market.IndustryGroupCode = instrument.Group
	}
	if instrument.GroupName != "" {
		market.IndustryGroupName = instrument.GroupName
	}
	if instrument.SubGroup != "" {
		market.IndustrySubCode = instrument.SubGroup
	}
	if instrument.InsertDate != nil {
		market.InsertDate = instrument.InsertDate
	}
	if instrument.AcceptanceDate != nil {
		market.AcceptanceDate = instrument.AcceptanceDate
	}
	if instrument.ReleaseDate != nil {
		market.IPODate = instrument.ReleaseDate
		market.IPODateSource = models.TSEIPODateSourceListing
	}
}

// SaveInfo stores the TSETMC instrument info of a market.
func (s *Service) SaveInfo(ctx context.Context, market *models.TSEMarket, info *tse.InstrumentInfo) error {
	if info == nil {
		return nil
	}
	if info.InsCode != market.InsCode {
		return fmt.Errorf("instrument info of %s does not belong to %s", info.InsCode, market.InsCode)
	}
	now := time.Now()
	columns := map[string]any{
		"company_isin":    info.CompanyISIN,
		"english_name":    info.EnglishName,
		"flow":            info.Flow,
		"board_code":      info.BoardCode,
		"total_shares":    info.TotalShares,
		"base_volume":     info.BaseVolume,
		"nav":             info.Nav,
		"eps":             info.EPS,
		"sector_pe":       info.SectorPE,
		"psr":             info.PSR,
		"info_updated_at": now,
	}
	if strings.HasPrefix(info.InstrumentID, "IR") {
		columns["instrument_id"] = info.InstrumentID
		market.InstrumentID = info.InstrumentID
	}
	if market.IndustryGroupCode == "" && info.SectorCode != "" {
		columns["industry_group_code"] = info.SectorCode
		columns["industry_group_name"] = info.SectorName
	}
	if err := s.tseRepo.Update(ctx, market.ID, columns); err != nil {
		return fmt.Errorf("save instrument info of %s: %w", market.InsCode, err)
	}
	market.CompanyISIN = info.CompanyISIN
	market.EnglishName = info.EnglishName
	market.Flow = info.Flow
	market.BoardCode = info.BoardCode
	market.TotalShares = info.TotalShares
	market.BaseVolume = info.BaseVolume
	market.Nav = info.Nav
	market.EPS = info.EPS
	market.SectorPE = info.SectorPE
	market.PSR = info.PSR
	market.InfoUpdatedAt = &now
	return nil
}

func (s *Service) ensureAsset(ctx context.Context, label, name string) (*models.Asset, error) {
	asset, err := s.assetRepo.ReturnByLabel(ctx, label)
	if err == nil {
		return asset, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	asset = &models.Asset{Label: label, Name: name, Type: consts.AssetStock}
	if err := s.assetRepo.Create(ctx, asset); err != nil {
		return nil, err
	}
	return asset, nil
}

// --- Panel ---

func (s *Service) HasAdapter() bool { return s.adapter != nil }

// List returns a page of TSE markets and the number of instruments linked to the
// market of each of them.
func (s *Service) List(ctx context.Context, filter repositories.TSEMarketFilter, page paginator.Params) ([]*models.TSEMarket, map[uint]int64, int64, error) {
	filter.Query = persianQuery(filter.Query)
	markets, total, err := s.tseRepo.Page(ctx, filter, page)
	if err != nil || len(markets) == 0 {
		return markets, map[uint]int64{}, total, err
	}
	marketIDs := make([]uint, len(markets))
	for i, market := range markets {
		marketIDs[i] = market.MarketID
	}
	counts, err := s.tseRepo.MemberCounts(ctx, marketIDs)
	return markets, counts, total, err
}

// MemberCount returns the number of instruments linked to a market.
func (s *Service) MemberCount(ctx context.Context, marketID uint) (int64, error) {
	counts, err := s.tseRepo.MemberCounts(ctx, []uint{marketID})
	return counts[marketID], err
}

func (s *Service) IndustryGroups(ctx context.Context) ([]repositories.TSEIndustryGroup, error) {
	return s.tseRepo.IndustryGroups(ctx)
}

// Flags are the panel flags; nil fields are left unchanged.
type Flags struct {
	DataCollectionEnabled *bool
	EvaluationEnabled     *bool
}

func (f Flags) columns() map[string]any {
	columns := map[string]any{}
	if f.DataCollectionEnabled != nil {
		columns["data_collection_enabled"] = *f.DataCollectionEnabled
	}
	if f.EvaluationEnabled != nil {
		columns["evaluation_enabled"] = *f.EvaluationEnabled
	}
	return columns
}

func (s *Service) SetFlags(ctx context.Context, id uint, flags Flags) (*models.TSEMarket, error) {
	columns := flags.columns()
	if len(columns) == 0 {
		return nil, fmt.Errorf("%w: no flag to change", ErrInvalidInput)
	}
	if _, err := s.tseRepo.SetFlags(ctx, []uint{id}, nil, columns); err != nil {
		return nil, err
	}
	s.logger().WithFields(logrus.Fields(columns)).WithField("id", id).Info("changed TSE market flags")
	s.scheduleDetection()
	return s.get(ctx, id)
}

// BulkSetFlags changes the flags of the given markets, or of every market that
// matches filter when ids is empty.
func (s *Service) BulkSetFlags(ctx context.Context, ids []uint, filter *repositories.TSEMarketFilter, flags Flags) (int64, error) {
	columns := flags.columns()
	if len(columns) == 0 {
		return 0, fmt.Errorf("%w: no flag to change", ErrInvalidInput)
	}
	if len(ids) == 0 && filter == nil {
		return 0, fmt.Errorf("%w: ids or filter is required", ErrInvalidInput)
	}
	if filter != nil {
		filter.Query = persianQuery(filter.Query)
	}
	updated, err := s.tseRepo.SetFlags(ctx, ids, filter, columns)
	if err == nil {
		s.logger().WithFields(logrus.Fields(columns)).WithFields(logrus.Fields{"ids": len(ids), "by_filter": len(ids) == 0, "updated": updated}).
			Info("changed TSE market flags in bulk")
		s.scheduleDetection()
	}
	return updated, err
}

// AddManual adds a symbol that is not stored yet, such as a new listing or the new
// insCode of a symbol that moved to another board. Its data is read from TSETMC
// and the TSE and IFB listing sources, and it joins the market of its symbol.
func (s *Service) AddManual(ctx context.Context, symbol, insCode string) (*models.TSEMarket, error) {
	symbol, insCode = strings.TrimSpace(symbol), strings.TrimSpace(insCode)
	if symbol == "" || insCode == "" {
		return nil, fmt.Errorf("%w: symbol and insCode are required", ErrInvalidInput)
	}
	if _, err := strconv.ParseUint(insCode, 10, 64); err != nil {
		return nil, fmt.Errorf("%w: insCode must be numeric", ErrInvalidInput)
	}
	if s.adapter == nil {
		return nil, ErrNoAdapter
	}
	if _, err := s.tseRepo.GetByInsCode(ctx, insCode); err == nil {
		return nil, fmt.Errorf("%w: insCode %s", ErrAlreadyExists, insCode)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	s.logger().WithFields(logrus.Fields{"symbol": symbol, "ins_code": insCode}).Info("adding TSE symbol by hand")
	instrument, err := s.adapter.ResolveInstrument(ctx, insCode, symbol)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	return s.storeOne(ctx, instrument, true)
}

// Refresh reads a stored instrument again from TSETMC and the listing sources.
func (s *Service) Refresh(ctx context.Context, id uint) (*models.TSEMarket, error) {
	if s.adapter == nil {
		return nil, ErrNoAdapter
	}
	market, err := s.get(ctx, id)
	if err != nil {
		return nil, err
	}
	s.logger().WithFields(logrus.Fields{"id": id, "symbol": market.Label, "ins_code": market.InsCode}).Info("refreshing TSE market")
	instrument, err := s.adapter.ResolveInstrument(ctx, market.InsCode, "")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	return s.storeOne(ctx, instrument, false)
}

func (s *Service) storeOne(ctx context.Context, instrument tse.Instrument, manual bool) (*models.TSEMarket, error) {
	state, err := s.loadSyncState(ctx)
	if err != nil {
		return nil, err
	}
	_, existed := state.byInsCode[instrument.InsCode]
	market, err := s.store(ctx, state, instrument, manual)
	if err != nil {
		return nil, err
	}
	if err := s.saveProblems(ctx, state, state.problems, time.Time{}); err != nil {
		return nil, err
	}
	s.logger().WithFields(logrus.Fields{
		"id": market.ID, "symbol": market.Label, "ins_code": market.InsCode, "market_id": market.MarketID,
		"created": !existed, "market_created": state.createdMarkets > 0, "problems": len(state.problems),
	}).Info("stored TSE market")
	s.scheduleDetection()
	return s.tseRepo.GetByID(ctx, market.ID)
}

// SetDelisted marks an instrument as delisted or listed. Catalog syncs never clear
// a delisted mark.
func (s *Service) SetDelisted(ctx context.Context, id uint, delisted bool) (*models.TSEMarket, error) {
	if _, err := s.get(ctx, id); err != nil {
		return nil, err
	}
	if err := s.tseRepo.Update(ctx, id, map[string]any{"delisted": delisted}); err != nil {
		return nil, err
	}
	s.logger().WithFields(logrus.Fields{"id": id, "delisted": delisted}).Info("changed TSE market delisted mark")
	s.scheduleDetection()
	return s.tseRepo.GetByID(ctx, id)
}

// Link moves an instrument to the market labelled label, or to a new market with
// that label. Instruments of a market must be of one kind. The link of an
// instrument whose prices are stored cannot change, because its candles belong
// to its current market.
func (s *Service) Link(ctx context.Context, id uint, label string) (*models.TSEMarket, error) {
	label = strings.TrimSpace(label)
	if tse.SymbolKey(label) == "" {
		return nil, fmt.Errorf("%w: market label is required", ErrInvalidInput)
	}
	tseMarket, err := s.get(ctx, id)
	if err != nil {
		return nil, err
	}
	if hasPrices, err := s.tseRepo.HasPrices(ctx, id); err != nil {
		return nil, err
	} else if hasPrices {
		return nil, fmt.Errorf("%w: prices of insCode %s are already stored in its market", ErrInvalidInput, tseMarket.InsCode)
	}
	state, err := s.loadSyncState(ctx)
	if err != nil {
		return nil, err
	}
	target := state.markets[tse.SymbolKey(label)]
	switch {
	case target == nil:
		trading, err := s.ensureAsset(ctx, tseMarket.Label, tseMarket.Name)
		if err != nil {
			return nil, fmt.Errorf("ensure trading asset: %w", err)
		}
		target = &models.Market{
			ProviderID:     state.providerID,
			PricingAssetID: state.destination.ID,
			TradingAssetID: trading.ID,
			Name:           tseMarket.Name,
			Label:          label,
			Status:         consts.MarketStatusOpen,
		}
	case target.ID == tseMarket.MarketID:
		return tseMarket, nil
	default:
		if kind := state.kinds[target.ID]; kind != "" && kind != tseMarket.Kind {
			return nil, fmt.Errorf("%w: market %q holds %s instruments", ErrInvalidInput, target.Label, kind.PersianLabel())
		}
	}
	newMarket := target.ID == 0
	previousDeleted, err := s.tseRepo.Link(ctx, id, target)
	if err != nil {
		return nil, fmt.Errorf("link TSE market %s to market %q: %w", tseMarket.InsCode, label, err)
	}
	s.logger().WithFields(logrus.Fields{
		"id": id, "ins_code": tseMarket.InsCode, "from_market_id": tseMarket.MarketID, "from_market_deleted": previousDeleted,
		"to_market_id": target.ID, "to_market": target.Label, "market_created": newMarket,
	}).Info("linked TSE market to another market")
	s.scheduleDetection()
	return s.tseRepo.GetByID(ctx, id)
}

// DeleteResult counts what Delete removed.
type DeleteResult struct {
	InsCodes int
	Markets  int
}

// Delete deletes an instrument with its prices, or with wholeMarket every
// instrument of its market. A market left without instruments is deleted with
// all of its data. Instruments that are in the TSE catalog are added again by
// the next catalog sync.
func (s *Service) Delete(ctx context.Context, id uint, wholeMarket bool) (DeleteResult, error) {
	var result DeleteResult
	market, err := s.get(ctx, id)
	if err != nil {
		return result, err
	}
	ids := []uint{id}
	if wholeMarket {
		members, err := s.tseRepo.List(ctx, map[string]any{"market_id = ?": market.MarketID})
		if err != nil {
			return result, err
		}
		ids = ids[:0]
		for _, member := range members {
			ids = append(ids, member.ID)
		}
	}
	deleted, err := s.tseRepo.Delete(ctx, ids)
	if err != nil {
		return result, fmt.Errorf("delete TSE market %s: %w", market.InsCode, err)
	}
	result = DeleteResult{InsCodes: len(deleted.InsCodes), Markets: len(deleted.MarketIDs)}
	resolved, err := s.tseRepo.ResolveIssuesOf(ctx, deleted.InsCodes)
	if err != nil {
		return result, fmt.Errorf("resolve issues of deleted insCodes: %w", err)
	}
	s.logger().WithFields(logrus.Fields{
		"id": id, "symbol": market.Label, "ins_codes": deleted.InsCodes, "market_id": market.MarketID,
		"whole_market": wholeMarket, "markets_deleted": deleted.MarketIDs, "issues_resolved": resolved,
	}).Info("deleted TSE market")
	s.scheduleDetection()
	return result, nil
}

func (s *Service) Get(ctx context.Context, id uint) (*models.TSEMarket, error) { return s.get(ctx, id) }

func (s *Service) get(ctx context.Context, id uint) (*models.TSEMarket, error) {
	market, err := s.tseRepo.GetByID(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return market, err
}

var persianQueryReplacer = strings.NewReplacer("ي", "ی", "ك", "ک")

func persianQuery(query string) string {
	return persianQueryReplacer.Replace(strings.TrimSpace(query))
}
