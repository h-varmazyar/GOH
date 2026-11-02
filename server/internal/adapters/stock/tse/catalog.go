package tse

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/sirupsen/logrus"
)

type catalogType struct {
	kind consts.TSEInstrumentKind
	// market is the board implied by the type; funds are placed by their flow.
	market consts.TSEMarketType
}

// catalogTypes are the TSETMC instrument types (yVal) returned by UpdateInstruments.
var catalogTypes = map[int]catalogType{
	300: {consts.TSEInstrumentKindStock, consts.TSEMarketTypeBourse},
	303: {consts.TSEInstrumentKindStock, consts.TSEMarketTypeIFB},
	309: {consts.TSEInstrumentKindStock, consts.TSEMarketTypeIFBBase},
	310: {consts.TSEInstrumentKindStock, consts.TSEMarketTypeIFBBase},
	313: {consts.TSEInstrumentKindStock, consts.TSEMarketTypeIFBNoAfarin},
	319: {consts.TSEInstrumentKindStock, consts.TSEMarketTypeIFBAgreement},
	305: {consts.TSEInstrumentKindFund, ""},
	400: {consts.TSEInstrumentKindRight, consts.TSEMarketTypeBourse},
	403: {consts.TSEInstrumentKindRight, consts.TSEMarketTypeIFB},
	404: {consts.TSEInstrumentKindRight, consts.TSEMarketTypeIFBBase},
	401: {consts.TSEInstrumentKindRight, consts.TSEMarketTypeIFBNoAfarin},
	329: {consts.TSEInstrumentKindRight, consts.TSEMarketTypeIFBAgreement},
}

// CatalogInstrumentTypes returns the TSETMC instrument types (yVal) that
// UpdateInstruments returns.
func CatalogInstrumentTypes() []int {
	types := make([]int, 0, len(catalogTypes))
	for value := range catalogTypes {
		types = append(types, value)
	}
	sort.Ints(types)
	return types
}

const (
	// ifbBoardStockType is the TSETMC type of the stocks of the IFB first and second markets.
	ifbBoardStockType = 303
	ifbBoardRightType = 403
	rightSymbolSuffix = "ح"

	tsetmcFlowBourse = 1
	tsetmcFlowIFB    = 2
	tsetmcMarketTSE  = 1

	// Names below this similarity are reported as a symbol and name mismatch.
	minNameSimilarity = 0.5

	infoFetchConcurrency = 4
	// catalogProgressEvery is how often progress of long catalog steps is logged.
	catalogProgressEvery = 250
)

type instrumentSearchResponse struct {
	InstrumentSearch []struct {
		InsCode string `json:"insCode"`
		Symbol  string `json:"lVal18AFC"`
		Name    string `json:"lVal30"`
	} `json:"instrumentSearch"`
}

type catalogEntry struct {
	localInstrument
	instrumentID string
	info         *InstrumentInfo
	marketType   consts.TSEMarketType
	board        *ifbBoardListing
	problems     []CatalogProblem
}

// CatalogProblem is something UpdateInstruments could not settle by itself and
// leaves for the user to decide on.
type CatalogProblem struct {
	Type    consts.TSEIssueType
	InsCode string
	Symbol  string
	Name    string
	Detail  string
}

// Catalog is the result of UpdateInstruments.
type Catalog struct {
	Instruments []Instrument
	Problems    []CatalogProblem
}

type catalogBuilder struct {
	adapter  *Adapter
	sources  *catalogSources
	logger   *logrus.Entry
	mu       sync.Mutex
	problems []CatalogProblem
}

func (b *catalogBuilder) report(problem CatalogProblem) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.problems = append(b.problems, problem)
}

func newCatalogBuilder(adapter *Adapter, sources *catalogSources) *catalogBuilder {
	return &catalogBuilder{adapter: adapter, sources: sources, logger: catalogLogger(adapter)}
}

func (b *catalogBuilder) build(ctx context.Context, local []localInstrument) (*Catalog, error) {
	entries := make([]*catalogEntry, 0, len(local))
	for _, item := range local {
		if _, ok := catalogTypes[item.Type]; !ok {
			continue
		}
		entries = append(entries, &catalogEntry{localInstrument: item})
	}
	b.logger.WithFields(logrus.Fields{"instruments": len(entries), "other_types": len(local) - len(entries)}).
		Info("selected stocks, funds and rights of the instrument files")
	entries, err := b.fetchInfo(ctx, entries)
	if err != nil {
		return nil, err
	}
	discovered, err := b.discover(ctx, entries)
	if err != nil {
		return nil, err
	}
	entries = append(entries, discovered...)
	instruments := b.classify(entries)
	for _, entry := range entries {
		b.problems = append(b.problems, entry.problems...)
	}
	for _, problem := range b.problems {
		b.logger.WithFields(logrus.Fields{"type": problem.Type, "ins_code": problem.InsCode, "symbol": problem.Symbol}).
			Warn(problem.Detail)
	}
	b.logSummary(instruments)
	return &Catalog{Instruments: instruments, Problems: b.problems}, nil
}

// logSummary logs the instruments by board and kind and the problems by type.
func (b *catalogBuilder) logSummary(instruments []Instrument) {
	byMarket, byKind, problems := logrus.Fields{}, logrus.Fields{}, logrus.Fields{}
	delisted := 0
	for _, instrument := range instruments {
		count, _ := byMarket[string(instrument.MarketType)].(int)
		byMarket[string(instrument.MarketType)] = count + 1
		count, _ = byKind[string(instrument.Kind)].(int)
		byKind[string(instrument.Kind)] = count + 1
		if instrument.Delisted {
			delisted++
		}
	}
	for _, problem := range b.problems {
		count, _ := problems[string(problem.Type)].(int)
		problems[string(problem.Type)] = count + 1
	}
	b.logger.WithFields(byMarket).WithField("instruments", len(instruments)).Info("catalog instruments by market type")
	b.logger.WithFields(byKind).WithFields(logrus.Fields{"delisted": delisted, "listed": len(instruments) - delisted}).
		Info("catalog instruments by kind")
	b.logger.WithFields(problems).WithField("problems", len(b.problems)).Info("catalog problems by type")
}

// fetchInfo fetches the TSETMC info of the entries. Entries whose info cannot be
// fetched are dropped and retried on the next run.
func (b *catalogBuilder) fetchInfo(ctx context.Context, entries []*catalogEntry) ([]*catalogEntry, error) {
	var (
		wg      sync.WaitGroup
		slots   = make(chan struct{}, infoFetchConcurrency)
		mu      sync.Mutex
		dropped = make(map[*catalogEntry]bool)
		done    atomic.Int64
		failed  atomic.Int64
		start   = time.Now()
	)
	b.logger.WithFields(logrus.Fields{"instruments": len(entries), "concurrency": infoFetchConcurrency}).
		Info("fetching TSETMC instrument info")
	progress := func() logrus.Fields {
		finished := done.Load()
		fields := logrus.Fields{
			"done": finished, "total": len(entries), "failed": failed.Load(), "elapsed": time.Since(start).Round(time.Second),
		}
		if finished > 0 && int(finished) < len(entries) {
			remaining := time.Duration(float64(time.Since(start)) / float64(finished) * float64(len(entries)-int(finished)))
			fields["remaining"] = remaining.Round(time.Second)
		}
		return fields
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			break
		}
		wg.Add(1)
		slots <- struct{}{}
		go func(entry *catalogEntry) {
			defer wg.Done()
			defer func() { <-slots }()
			defer func() {
				if finished := done.Add(1); finished%catalogProgressEvery == 0 {
					b.logger.WithFields(progress()).Info("fetching TSETMC instrument info")
				}
			}()
			info, err := b.adapter.FetchInstrumentInfo(ctx, entry.InsCode)
			if err == nil {
				err = entry.applyInfo(info)
			}
			if err != nil {
				failed.Add(1)
				if ctx.Err() == nil {
					b.logger.WithError(err).WithFields(logrus.Fields{"ins_code": entry.InsCode, "symbol": entry.Symbol}).
						Warn("skip TSE instrument whose TSETMC info cannot be read")
					b.report(CatalogProblem{
						Type: consts.TSEIssueSkipped, InsCode: entry.InsCode, Symbol: entry.Symbol, Name: cleanText(entry.Name),
						Detail: fmt.Sprintf("اطلاعات این insCode از TSETMC خوانده نشد و نماد ذخیره نشد: %v", err),
					})
				}
				mu.Lock()
				dropped[entry] = true
				mu.Unlock()
			}
		}(entry)
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	result := entries[:0]
	for _, entry := range entries {
		if !dropped[entry] {
			result = append(result, entry)
		}
	}
	b.logger.WithFields(progress()).WithField("kept", len(result)).Info("fetched TSETMC instrument info")
	return result, nil
}

// applyInfo takes the instrument ID from TSETMC and checks that the instrument
// still has the type and symbol of the instrument files.
func (e *catalogEntry) applyInfo(info *InstrumentInfo) error {
	if info.InsCode != e.InsCode {
		return fmt.Errorf("TSETMC returned insCode %s", info.InsCode)
	}
	if !strings.HasPrefix(info.InstrumentID, "IR") {
		return fmt.Errorf("TSETMC returned no instrument ID")
	}
	if info.Type != 0 && info.Type != e.Type {
		if _, ok := catalogTypes[info.Type]; !ok {
			return fmt.Errorf("TSETMC instrument type changed from %d to %d", e.Type, info.Type)
		}
		e.Type = info.Type
	}
	if info.Symbol != "" && matchKey(info.Symbol) != matchKey(e.Symbol) {
		e.problems = append(e.problems, CatalogProblem{
			Type: consts.TSEIssueSymbolChanged, InsCode: e.InsCode, Symbol: cleanText(info.Symbol), Name: cleanText(info.Name),
			Detail: fmt.Sprintf("نماد این insCode در فایل‌ها «%s» و در TSETMC «%s» است؛ نماد TSETMC ذخیره شد.", e.Symbol, cleanText(info.Symbol)),
		})
		e.Symbol = cleanText(info.Symbol)
	}
	if e.Group == "" {
		e.Group = info.SectorCode
	}
	e.instrumentID = info.InstrumentID
	e.info = info
	return nil
}

// discoveryWant is a symbol announced by a listing source but missing from the
// catalog. instrumentID is empty for IFB board symbols.
type discoveryWant struct {
	symbol       string
	instrumentID string
}

// discovery finds instruments that the listing sources announce but the instrument
// files do not have. TSE listings are verified by instrument ID; IFB board symbols
// by TSETMC type and symbol.
type discovery struct {
	builder       *catalogBuilder
	knownInsCodes map[string]bool
	searched      map[string][]string
	infos         map[string]*InstrumentInfo
}

func (b *catalogBuilder) discover(ctx context.Context, entries []*catalogEntry) ([]*catalogEntry, error) {
	d := &discovery{
		builder:       b,
		knownInsCodes: make(map[string]bool, len(entries)),
		searched:      make(map[string][]string),
		infos:         make(map[string]*InstrumentInfo),
	}
	for _, entry := range entries {
		d.knownInsCodes[entry.InsCode] = true
	}
	wants := b.discoveryWants(entries)
	byID := 0
	for _, want := range wants {
		if want.instrumentID != "" {
			byID++
		}
	}
	b.logger.WithFields(logrus.Fields{"listings": byID, "ifb_board_symbols": len(wants) - byID}).
		Info("searching TSETMC for listed symbols missing from the instrument files")
	start := time.Now()
	result := make([]*catalogEntry, 0)
	for i, want := range wants {
		found, err := d.resolve(ctx, want)
		if err != nil {
			return nil, err
		}
		result = append(result, found...)
		if (i+1)%catalogProgressEvery == 0 {
			b.logger.WithFields(logrus.Fields{"done": i + 1, "total": len(wants), "found": len(result)}).
				Info("searching TSETMC for missing symbols")
		}
	}
	b.logger.WithFields(logrus.Fields{
		"searched": len(wants), "symbols": len(d.searched), "found": len(result), "duration": time.Since(start).Round(time.Second),
	}).Info("searched TSETMC for missing symbols")
	return result, nil
}

func (b *catalogBuilder) discoveryWants(entries []*catalogEntry) []discoveryWant {
	knownIDs := make(map[string]bool, len(entries))
	activeBoardStocks := make(map[string]bool)
	for _, entry := range entries {
		knownIDs[entry.instrumentID] = true
		if entry.Type == ifbBoardStockType && entry.Active {
			activeBoardStocks[matchKey(entry.Symbol)] = true
		}
	}
	ids := make([]string, 0, len(b.sources.byID))
	for id := range b.sources.byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	wants := make([]discoveryWant, 0)
	for _, id := range ids {
		if listing := b.sources.byID[id]; !knownIDs[id] && listing.Symbol != "" {
			wants = append(wants, discoveryWant{symbol: listing.Symbol, instrumentID: id})
		}
	}
	for _, row := range b.sources.ifbBoards {
		if !activeBoardStocks[matchKey(row.Symbol)] {
			wants = append(wants, discoveryWant{symbol: row.Symbol})
		}
	}
	return wants
}

// resolve searches TSETMC for the symbol of want and returns the new instruments
// that match it. Only context errors are returned; other failures are logged.
func (d *discovery) resolve(ctx context.Context, want discoveryWant) ([]*catalogEntry, error) {
	key := matchKey(want.symbol)
	insCodes, ok := d.searched[key]
	if !ok {
		var err error
		if insCodes, err = d.builder.search(ctx, want.symbol); err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			d.builder.logger.WithError(err).WithField("symbol", want.symbol).Warn("search TSETMC instrument")
		}
		d.searched[key] = insCodes
	}
	result := make([]*catalogEntry, 0)
	for _, insCode := range insCodes {
		if d.knownInsCodes[insCode] {
			continue
		}
		info, err := d.info(ctx, insCode)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			d.builder.logger.WithError(err).WithField("ins_code", insCode).Warn("fetch TSETMC instrument info")
			continue
		}
		if !want.matches(info) {
			continue
		}
		d.knownInsCodes[insCode] = true
		result = append(result, &catalogEntry{
			localInstrument: localInstrument{
				InsCode:      insCode,
				Symbol:       cleanText(info.Symbol),
				Name:         info.Name,
				Group:        info.SectorCode,
				Type:         info.Type,
				SymbolStatus: "A",
				Active:       true,
			},
			instrumentID: info.InstrumentID,
			info:         info,
		})
		d.builder.logger.WithFields(logrus.Fields{"ins_code": insCode, "symbol": info.Symbol, "instrument_id": info.InstrumentID}).
			Info("found TSE instrument missing from the instrument files")
	}
	return result, nil
}

func (d *discovery) info(ctx context.Context, insCode string) (*InstrumentInfo, error) {
	if info, ok := d.infos[insCode]; ok {
		return info, nil
	}
	info, err := d.builder.adapter.FetchInstrumentInfo(ctx, insCode)
	if err != nil {
		return nil, err
	}
	d.infos[insCode] = info
	return info, nil
}

func (w discoveryWant) matches(info *InstrumentInfo) bool {
	if _, ok := catalogTypes[info.Type]; !ok || !strings.HasPrefix(info.InstrumentID, "IR") {
		return false
	}
	if w.instrumentID != "" {
		return info.InstrumentID == w.instrumentID
	}
	return info.Type == ifbBoardStockType && matchKey(info.Symbol) == matchKey(w.symbol)
}

func (b *catalogBuilder) search(ctx context.Context, symbol string) ([]string, error) {
	var response instrumentSearchResponse
	if err := b.adapter.fetchJSON(ctx, fmt.Sprintf(fetchInstrumentSearchURL, url.PathEscape(symbol)), &response); err != nil {
		return nil, err
	}
	key := matchKey(symbol)
	result := make([]string, 0)
	for _, item := range response.InstrumentSearch {
		if matchKey(item.Symbol) == key && strings.TrimSpace(item.InsCode) != "" {
			result = append(result, strings.TrimSpace(item.InsCode))
		}
	}
	return result, nil
}

// classify sets the board, dates and names of the entries. The returned slice puts
// active and traded instruments first, so they win a symbol shared with others.
func (b *catalogBuilder) classify(entries []*catalogEntry) []Instrument {
	for _, entry := range entries {
		entry.marketType = entry.defaultMarketType()
	}
	b.assignIFBBoards(entries)

	sort.SliceStable(entries, func(i, j int) bool { return entries[i].rankBefore(entries[j]) })
	result := make([]Instrument, 0, len(entries))
	for _, entry := range entries {
		result = append(result, b.instrument(entry))
	}
	return result
}

func (e *catalogEntry) defaultMarketType() consts.TSEMarketType {
	definition := catalogTypes[e.Type]
	if definition.kind != consts.TSEInstrumentKindFund {
		return definition.market
	}
	switch {
	case strings.HasPrefix(e.instrumentID, "IRT1"):
		return consts.TSEMarketTypeBourse
	case strings.HasPrefix(e.instrumentID, "IRT3"):
		return consts.TSEMarketTypeIFB
	case e.info != nil && e.info.Flow == tsetmcFlowBourse:
		return consts.TSEMarketTypeBourse
	case e.info != nil && e.info.Flow == tsetmcFlowIFB:
		return consts.TSEMarketTypeIFB
	case e.MarketCode == tsetmcMarketTSE:
		return consts.TSEMarketTypeBourse
	default:
		return consts.TSEMarketTypeIFB
	}
}

// rankBefore orders active instruments before delisted ones and traded symbols
// (status "A") before symbols that have not been traded yet ("IS").
func (e *catalogEntry) rankBefore(other *catalogEntry) bool {
	if e.Active != other.Active {
		return e.Active
	}
	if traded, otherTraded := e.SymbolStatus == "A", other.SymbolStatus == "A"; traded != otherTraded {
		return traded
	}
	return false
}

// assignIFBBoards marks the active IFB stocks listed on the IFB first or second
// market; rights of those stocks follow their stock.
func (b *catalogBuilder) assignIFBBoards(entries []*catalogEntry) {
	stocks := make(map[string][]*catalogEntry)
	for _, entry := range entries {
		if entry.Type == ifbBoardStockType && entry.Active {
			key := matchKey(entry.Symbol)
			stocks[key] = append(stocks[key], entry)
		}
	}
	for i := range b.sources.ifbBoards {
		row := &b.sources.ifbBoards[i]
		candidates := stocks[matchKey(row.Symbol)]
		var best *catalogEntry
		bestSimilarity := -1.0
		for _, candidate := range candidates {
			if candidate.board != nil {
				continue
			}
			similarity := nameSimilarity(row.Name, candidate.Name)
			if best == nil || candidate.rankBefore(best) || (!best.rankBefore(candidate) && similarity > bestSimilarity) {
				best, bestSimilarity = candidate, similarity
			}
		}
		if best == nil {
			b.report(CatalogProblem{
				Type: consts.TSEIssueIFBBoardUnmatched, Symbol: row.Symbol, Name: row.Name,
				Detail: fmt.Sprintf("نماد «%s» در %s فرابورس فهرست شده ولی سهم فعالی با این نماد در TSETMC پیدا نشد.", row.Symbol, row.MarketType.PersianLabel()),
			})
			continue
		}
		if row.Name != "" && bestSimilarity < minNameSimilarity {
			b.report(CatalogProblem{
				Type: consts.TSEIssueIFBNameMismatch, InsCode: best.InsCode, Symbol: row.Symbol, Name: row.Name,
				Detail: fmt.Sprintf("نام فرابورس «%s» با نام TSETMC «%s» شباهت %.0f%% دارد؛ بازار %s برای این insCode ثبت شد.",
					row.Name, cleanText(best.Name), 100*bestSimilarity, row.MarketType.PersianLabel()),
			})
		}
		best.board = row
		best.marketType = row.MarketType
	}

	boardOf := make(map[string]consts.TSEMarketType)
	for _, entry := range entries {
		if entry.board != nil {
			boardOf[matchKey(entry.Symbol)] = entry.marketType
		}
	}
	for _, entry := range entries {
		if entry.Type != ifbBoardRightType {
			continue
		}
		parent := strings.TrimSuffix(matchKey(entry.Symbol), rightSymbolSuffix)
		if marketType, ok := boardOf[parent]; ok {
			entry.marketType = marketType
		}
	}
}

func (b *catalogBuilder) instrument(entry *catalogEntry) Instrument {
	instrument := Instrument{
		InsCode:        entry.InsCode,
		InstrumentID:   entry.instrumentID,
		Symbol:         entry.Symbol,
		Name:           cleanText(entry.Name),
		MarketType:     entry.marketType,
		Kind:           catalogTypes[entry.Type].kind,
		InstrumentType: entry.Type,
		Group:          strings.TrimSpace(entry.Group),
		SubGroup:       entry.SubGroup,
		Delisted:       !entry.Active,
		Info:           entry.info,
	}
	if entry.info != nil && entry.info.Name != "" {
		instrument.Name = cleanText(entry.info.Name)
	}
	listing, listed := b.sources.byID[entry.instrumentID]
	if listed {
		instrument.InsertDate = listing.InsertDate
		instrument.ReleaseDate = listing.ReleaseDate
		if listing.Name != "" {
			instrument.Name = listing.Name
		}
	}
	if board := entry.board; board != nil {
		if board.InsertDate != nil {
			instrument.InsertDate = board.InsertDate
		}
		if board.ReleaseDate != nil {
			instrument.ReleaseDate = board.ReleaseDate
		}
		instrument.AcceptanceDate = board.AcceptanceDate
		if !listed && board.Name != "" && nameSimilarity(board.Name, entry.Name) >= minNameSimilarity {
			instrument.Name = board.Name
		}
	}
	if name := b.sources.groups[groupKey(instrument.Group)]; name != "" {
		instrument.GroupName = name
	} else if entry.info != nil {
		instrument.GroupName = entry.info.SectorName
	}
	return instrument
}
