package tsemarket

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/h-varmazyar/gatec/internal/adapters/stock/tse"
	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/h-varmazyar/gatec/pkg/paginator"
	"github.com/sirupsen/logrus"
)

// detectionTimeout bounds checks that run after panel changes.
const detectionTimeout = 5 * time.Minute

var problemSeverity = map[consts.TSEIssueType]consts.TSEIssueSeverity{
	consts.TSEIssueSkipped:           consts.TSEIssueSeverityError,
	consts.TSEIssueKindConflict:      consts.TSEIssueSeverityWarning,
	consts.TSEIssueLabelConflict:     consts.TSEIssueSeverityWarning,
	consts.TSEIssueIFBBoardUnmatched: consts.TSEIssueSeverityWarning,
	consts.TSEIssueIFBNameMismatch:   consts.TSEIssueSeverityWarning,
	consts.TSEIssueSymbolChanged:     consts.TSEIssueSeverityInfo,
}

// saveProblems stores problems found while instruments were stored. With
// resolveBefore set, open problems of earlier syncs not found again are resolved.
func (s *Service) saveProblems(ctx context.Context, state *syncState, problems []tse.CatalogProblem, resolveBefore time.Time) error {
	now := time.Now()
	issues := make([]*models.TSEMarketIssue, 0, len(problems))
	seen := make(map[string]bool, len(problems))
	for _, problem := range problems {
		key := fmt.Sprintf("%s:%s", problem.Type, problem.InsCode)
		if problem.InsCode == "" {
			key = fmt.Sprintf("%s:%s", problem.Type, tse.SymbolKey(problem.Symbol))
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		issue := &models.TSEMarketIssue{
			Key:        key,
			Type:       problem.Type,
			Severity:   problemSeverity[problem.Type],
			Status:     consts.TSEIssueStatusOpen,
			Symbol:     problem.Symbol,
			Name:       problem.Name,
			Detail:     problem.Detail,
			InsCodes:   []string{},
			MarketIDs:  []uint{},
			DetectedAt: now,
		}
		if problem.InsCode != "" {
			issue.InsCodes = []string{problem.InsCode}
			if market := state.byInsCode[problem.InsCode]; market != nil {
				issue.MarketIDs = []uint{market.MarketID}
			}
		}
		issues = append(issues, issue)
	}
	if err := s.tseRepo.SaveIssues(ctx, issues); err != nil {
		return fmt.Errorf("save catalog issues: %w", err)
	}
	logger := s.logger().WithField("issues", len(issues))
	if resolveBefore.IsZero() {
		if len(issues) > 0 {
			logger.Info("saved catalog issues")
		}
		return nil
	}
	resolved, err := s.tseRepo.ResolveStaleIssues(ctx, consts.TSEIssueSyncTypes, resolveBefore)
	if err != nil {
		return fmt.Errorf("resolve catalog issues: %w", err)
	}
	logger.WithField("resolved", resolved).Info("saved catalog issues; resolved the ones of earlier syncs not found again")
	return nil
}

// DetectionResult counts the issues found by a check of the stored markets.
type DetectionResult struct {
	Found    int
	Resolved int64
}

// DetectIssues checks the stored markets for insCodes that may be linked to the
// wrong market, by symbol and by company name, and for insCodes whose prices
// share candles. Issues that are no longer found are resolved.
func (s *Service) DetectIssues(ctx context.Context) (DetectionResult, error) {
	s.detectMu.Lock()
	defer s.detectMu.Unlock()

	var result DetectionResult
	logger := s.logger()
	start := time.Now()
	logger.Info("checking TSE markets for issues")
	markets, err := s.tseRepo.List(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("list TSE markets: %w", err)
	}
	overlaps, err := s.tseRepo.PriceOverlaps(ctx)
	if err != nil {
		return result, fmt.Errorf("find price overlaps: %w", err)
	}
	check := newIssueCheck(markets, start)
	check.nameMismatches()
	check.sameNames()
	check.multipleActive()
	check.priceOverlaps(overlaps)
	if err := s.tseRepo.SaveIssues(ctx, check.issues); err != nil {
		return result, fmt.Errorf("save issues: %w", err)
	}
	result.Found = len(check.issues)

	resolved, err := s.tseRepo.ResolveStaleIssues(ctx, consts.TSEIssueCheckTypes, start)
	if err != nil {
		return result, fmt.Errorf("resolve issues: %w", err)
	}
	result.Resolved = resolved
	added, err := s.resolveAddedSkipped(ctx, check.byInsCode)
	result.Resolved += added
	if err != nil {
		return result, err
	}
	byType := logrus.Fields{}
	for _, issue := range check.issues {
		count, _ := byType[string(issue.Type)].(int)
		byType[string(issue.Type)] = count + 1
	}
	logger.WithFields(byType).WithFields(logrus.Fields{
		"tse_markets": len(markets), "markets": len(check.marketIDs), "found": result.Found,
		"resolved": result.Resolved, "duration": time.Since(start).Round(time.Millisecond),
	}).Info("checked TSE markets for issues")
	return result, nil
}

// resolveAddedSkipped resolves skipped instruments that were added since.
func (s *Service) resolveAddedSkipped(ctx context.Context, byInsCode map[string]*models.TSEMarket) (int64, error) {
	skipped, err := s.tseRepo.ListIssues(ctx, repositories.TSEIssueFilter{Type: consts.TSEIssueSkipped, Status: consts.TSEIssueStatusOpen})
	if err != nil {
		return 0, err
	}
	ids := make([]uint, 0)
	for _, issue := range skipped {
		if len(issue.InsCodes) > 0 && byInsCode[issue.InsCodes[0]] != nil {
			ids = append(ids, issue.ID)
		}
	}
	if len(ids) == 0 {
		return 0, nil
	}
	now := time.Now()
	return s.tseRepo.SetIssueStatus(ctx, ids, nil, map[string]any{
		"status": consts.TSEIssueStatusResolved, "resolved_at": now, "updated_at": now,
	})
}

// scheduleDetection checks the stored markets in the background after a panel
// change. Changes made while a check runs are checked once more afterwards.
func (s *Service) scheduleDetection() {
	s.detectState.Lock()
	if s.detectRunning {
		s.detectAgain = true
		s.detectState.Unlock()
		return
	}
	s.detectRunning = true
	s.detectState.Unlock()
	go func() {
		for {
			ctx, cancel := context.WithTimeout(context.Background(), detectionTimeout)
			if _, err := s.DetectIssues(ctx); err != nil {
				s.logger().WithError(err).Error("check TSE markets for issues")
			}
			cancel()
			s.detectState.Lock()
			if !s.detectAgain {
				s.detectRunning = false
				s.detectState.Unlock()
				return
			}
			s.detectAgain = false
			s.detectState.Unlock()
		}
	}()
}

type issueCheck struct {
	now       time.Time
	byInsCode map[string]*models.TSEMarket
	byMarket  map[uint][]*models.TSEMarket
	marketIDs []uint
	issues    []*models.TSEMarketIssue
}

func newIssueCheck(markets []*models.TSEMarket, now time.Time) *issueCheck {
	check := &issueCheck{
		now:       now,
		byInsCode: make(map[string]*models.TSEMarket, len(markets)),
		byMarket:  make(map[uint][]*models.TSEMarket),
	}
	for _, market := range markets {
		check.byInsCode[market.InsCode] = market
		if _, ok := check.byMarket[market.MarketID]; !ok {
			check.marketIDs = append(check.marketIDs, market.MarketID)
		}
		check.byMarket[market.MarketID] = append(check.byMarket[market.MarketID], market)
	}
	slices.Sort(check.marketIDs)
	return check
}

func (c *issueCheck) add(issueType consts.TSEIssueType, severity consts.TSEIssueSeverity, key string, members []*models.TSEMarket, marketIDs []uint, detail string) {
	insCodes := make([]string, len(members))
	for i, member := range members {
		insCodes[i] = member.InsCode
	}
	c.issues = append(c.issues, &models.TSEMarketIssue{
		Key:        fmt.Sprintf("%s:%s", issueType, key),
		Type:       issueType,
		Severity:   severity,
		Status:     consts.TSEIssueStatusOpen,
		Symbol:     members[0].Label,
		Name:       members[0].Name,
		Detail:     detail,
		InsCodes:   insCodes,
		MarketIDs:  marketIDs,
		DetectedAt: c.now,
	})
}

func marketName(members []*models.TSEMarket) (label, name string) {
	if market := members[0].Market; market != nil {
		return market.Label, market.Name
	}
	return members[0].Label, members[0].Name
}

func insCodeKey(members []*models.TSEMarket) string {
	insCodes := make([]string, len(members))
	for i, member := range members {
		insCodes[i] = member.InsCode
	}
	slices.Sort(insCodes)
	return strings.Join(insCodes, ",")
}

// nameMismatches finds insCodes whose name differs from the name of their market.
// When a differing insCode and another insCode are both listed they are likely
// two companies that share a symbol; otherwise the company was likely renamed.
func (c *issueCheck) nameMismatches() {
	for _, marketID := range c.marketIDs {
		members := c.byMarket[marketID]
		if len(members) < 2 {
			continue
		}
		label, name := marketName(members)
		differing := make([]*models.TSEMarket, 0)
		names := make([]string, 0)
		for _, member := range members {
			if similarity := tse.NameSimilarity(member.Name, name); similarity < minMemberNameSimilarity {
				differing = append(differing, member)
				names = append(names, fmt.Sprintf("«%s» (%s، شباهت %.0f%%)", member.Name, member.InsCode, 100*similarity))
			}
		}
		if len(differing) == 0 {
			continue
		}
		severity := consts.TSEIssueSeverityWarning
		for _, member := range differing {
			if !member.Delisted && countListed(members) > 1 {
				severity = consts.TSEIssueSeverityError
			}
		}
		c.add(consts.TSEIssueNameMismatch, severity, fmt.Sprintf("%d:%s", marketID, insCodeKey(differing)), differing, []uint{marketID},
			fmt.Sprintf("نام این insCodeها با نام مارکت «%s» («%s») متفاوت است: %s. اگر شرکت دیگری‌اند به مارکت جداگانه منتقلشان کنید؛ اگر همان شرکت با نام قبلی‌اند تأیید کنید.",
				label, name, strings.Join(names, "، ")))
	}
}

func countListed(members []*models.TSEMarket) int {
	count := 0
	for _, member := range members {
		if !member.Delisted {
			count++
		}
	}
	return count
}

// sameNames finds markets of different symbols whose insCodes have the same
// company name and kind, such as a symbol renamed on moving between boards.
func (c *issueCheck) sameNames() {
	type nameGroup struct {
		members   []*models.TSEMarket
		marketIDs []uint
	}
	groups := make(map[string]*nameGroup)
	keys := make([]string, 0)
	for _, marketID := range c.marketIDs {
		for _, member := range c.byMarket[marketID] {
			nameKey := tse.CompanyNameKey(member.Name)
			if nameKey == "" {
				continue
			}
			key := string(member.Kind) + ":" + nameKey
			group := groups[key]
			if group == nil {
				group = &nameGroup{}
				groups[key] = group
				keys = append(keys, key)
			}
			group.members = append(group.members, member)
			if !slices.Contains(group.marketIDs, marketID) {
				group.marketIDs = append(group.marketIDs, marketID)
			}
		}
	}
	for _, key := range keys {
		group := groups[key]
		if len(group.marketIDs) < 2 {
			continue
		}
		labels := make([]string, 0, len(group.marketIDs))
		ids := make([]string, 0, len(group.marketIDs))
		for _, marketID := range group.marketIDs {
			label, _ := marketName(c.byMarket[marketID])
			labels = append(labels, "«"+label+"»")
			ids = append(ids, fmt.Sprint(marketID))
		}
		c.add(consts.TSEIssueSameName, consts.TSEIssueSeverityWarning, strings.Join(ids, ","), group.members, group.marketIDs,
			fmt.Sprintf("مارکت‌های %s نام شرکت یکسان «%s» دارند. اگر یک شرکت‌اند (مثلاً تغییر نماد هنگام انتقال بین بازارها) در یک مارکت ادغامشان کنید، وگرنه تأیید کنید.",
				strings.Join(labels, "، "), group.members[0].Name))
	}
}

// multipleActive finds markets with several listed insCodes that collect data.
// Their prices of the same day would share one candle.
func (c *issueCheck) multipleActive() {
	for _, marketID := range c.marketIDs {
		active := make([]*models.TSEMarket, 0)
		for _, member := range c.byMarket[marketID] {
			if !member.Delisted && member.DataCollectionEnabled {
				active = append(active, member)
			}
		}
		if len(active) < 2 {
			continue
		}
		label, _ := marketName(c.byMarket[marketID])
		c.add(consts.TSEIssueMultipleActive, consts.TSEIssueSeverityInfo, fmt.Sprintf("%d:%s", marketID, insCodeKey(active)), active, []uint{marketID},
			fmt.Sprintf("%d insCode فعال در مارکت «%s» داده دریافت می‌کنند. اگر در یک روز هر دو معامله شوند، کندل آن روز فقط از یکی ساخته می‌شود. insCodeی که دیگر معامله نمی‌شود را لغو پذیرش کنید یا دریافت داده‌اش را خاموش کنید.",
				len(active), label))
	}
}

func (c *issueCheck) priceOverlaps(overlaps []repositories.TSEPriceOverlap) {
	for _, overlap := range overlaps {
		members := make([]*models.TSEMarket, 0)
		for _, insCode := range strings.Split(overlap.InsCodes, ",") {
			if member := c.byInsCode[insCode]; member != nil {
				members = append(members, member)
			}
		}
		if len(members) == 0 {
			continue
		}
		label, _ := marketName(c.byMarket[overlap.MarketID])
		c.add(consts.TSEIssuePriceOverlap, consts.TSEIssueSeverityError, fmt.Sprintf("%d:%s", overlap.MarketID, overlap.InsCodes), members, []uint{overlap.MarketID},
			fmt.Sprintf("insCodeهای مارکت «%s» در %d روز هم‌زمان قیمت دارند و کندل آن روزها فقط از یکی از آن‌هاست. دریافت داده insCode نادرست را خاموش کنید.",
				label, overlap.Days))
	}
}

// --- Panel ---

// IssueView is an issue with the stored insCodes it is about and the other
// insCodes of its markets.
type IssueView struct {
	Issue       *models.TSEMarketIssue
	Instruments []*models.TSEMarket
}

func (s *Service) ListIssues(ctx context.Context, filter repositories.TSEIssueFilter, page paginator.Params) ([]IssueView, map[uint]int64, int64, error) {
	filter.Query = persianQuery(filter.Query)
	issues, total, err := s.tseRepo.PageIssues(ctx, filter, page)
	if err != nil {
		return nil, nil, 0, err
	}
	insCodes := make([]string, 0)
	marketIDs := make([]uint, 0)
	for _, issue := range issues {
		insCodes = append(insCodes, issue.InsCodes...)
		marketIDs = append(marketIDs, issue.MarketIDs...)
	}
	related, err := s.tseRepo.ListRelated(ctx, insCodes, marketIDs)
	if err != nil {
		return nil, nil, 0, err
	}
	views := make([]IssueView, len(issues))
	for i, issue := range issues {
		views[i] = IssueView{Issue: issue, Instruments: make([]*models.TSEMarket, 0)}
		for _, market := range related {
			if slices.Contains(issue.InsCodes, market.InsCode) || slices.Contains(issue.MarketIDs, market.MarketID) {
				views[i].Instruments = append(views[i].Instruments, market)
			}
		}
	}
	memberMarkets := make([]uint, 0, len(related))
	for _, market := range related {
		memberMarkets = append(memberMarkets, market.MarketID)
	}
	counts := map[uint]int64{}
	if len(memberMarkets) > 0 {
		if counts, err = s.tseRepo.MemberCounts(ctx, memberMarkets); err != nil {
			return nil, nil, 0, err
		}
	}
	return views, counts, total, nil
}

// IssueSummary counts issues by type, severity and status.
func (s *Service) IssueSummary(ctx context.Context) ([]repositories.TSEIssueCount, error) {
	return s.tseRepo.IssueCounts(ctx)
}

// SetIssueStatus opens or ignores the given issues, or every issue matching filter
// when ids is empty. Ignoring records that the user accepts the situation.
func (s *Service) SetIssueStatus(ctx context.Context, ids []uint, filter *repositories.TSEIssueFilter, status consts.TSEIssueStatus, note *string) (int64, error) {
	if status != consts.TSEIssueStatusOpen && status != consts.TSEIssueStatusIgnored {
		return 0, fmt.Errorf("%w: status must be %s or %s", ErrInvalidInput, consts.TSEIssueStatusOpen, consts.TSEIssueStatusIgnored)
	}
	if len(ids) == 0 && filter == nil {
		return 0, fmt.Errorf("%w: ids or filter is required", ErrInvalidInput)
	}
	if filter != nil {
		filter.Query = persianQuery(filter.Query)
	}
	columns := map[string]any{"status": status, "resolved_at": nil, "updated_at": time.Now()}
	if note != nil {
		columns["note"] = strings.TrimSpace(*note)
	}
	updated, err := s.tseRepo.SetIssueStatus(ctx, ids, filter, columns)
	if err == nil {
		s.logger().WithFields(logrus.Fields{"status": status, "ids": len(ids), "by_filter": len(ids) == 0, "updated": updated}).
			Info("changed TSE issue status")
	}
	return updated, err
}
