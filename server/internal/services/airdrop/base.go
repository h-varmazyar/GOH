package airdrop

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/url"
	"regexp"
	"strings"

	"github.com/h-varmazyar/gatec/configs"
	"github.com/h-varmazyar/gatec/internal/adapters/crypto/tabdeal"
	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/h-varmazyar/gatec/internal/services/tgaccount"
	"github.com/h-varmazyar/gatec/pkg/paginator"
	"gorm.io/gorm"
)

const (
	GiftStatusInitiated        = "initiated"
	GiftStatusReceived         = "received"
	GiftStatusPartialyReceived = "partialy_received"
	GiftStatusFailed           = "failed"
	GiftStatusRouteNotFound    = "route_not_found"
)

type AdminNotifier interface {
	NotifyAdmin(ctx context.Context, text string)
}

type Service struct {
	tabdealAirdropRepo    *repositories.TabdealAirdropRepository
	tabdealAllocationRepo *repositories.TabdealAllocationRepository
	UserRepo              *repositories.UserRepository
	tabdeal               *tabdeal.Adapter
	tgAccountService      *tgaccount.Service
}

func NewService(
	cfg configs.AirdropService,
	tabdealAirdropRepo *repositories.TabdealAirdropRepository,
	tabdealAllocationRepo *repositories.TabdealAllocationRepository,
	userRepo *repositories.UserRepository,
	tabdealAdapter *tabdeal.Adapter,
	tgAccountService *tgaccount.Service,
) *Service {
	return &Service{
		tabdealAirdropRepo:    tabdealAirdropRepo,
		tabdealAllocationRepo: tabdealAllocationRepo,
		UserRepo:              userRepo,
		tabdeal:               tabdealAdapter,
		tgAccountService:      tgAccountService,
	}
}

func (s *Service) CreateRoute(ctx context.Context, phrase, route string) (*models.TabdealAirdrop, error) {
	item := &models.TabdealAirdrop{Phrase: strings.TrimSpace(phrase), Route: strings.TrimSpace(route)}
	if item.Phrase == "" || item.Route == "" {
		return nil, fmt.Errorf("phrase and route are required")
	}
	return item, s.tabdealAirdropRepo.UpsertRoute(ctx, item)
}

func (s *Service) ListRoutes(ctx context.Context, pg paginator.Params) ([]models.TabdealAirdrop, int64, error) {
	return s.tabdealAirdropRepo.ListRoutes(ctx, pg)
}
func (s *Service) ListGifts(ctx context.Context, userID uint, pg paginator.Params) ([]models.TabdealAirdropGift, int64, error) {
	return s.tabdealAirdropRepo.ListGifts(ctx, userID, pg)
}
func (s *Service) GetGift(ctx context.Context, userID, id uint) (*models.TabdealAirdropGift, error) {
	return s.tabdealAirdropRepo.GetUserGift(ctx, userID, id)
}

func (s *Service) RetryCollectGiftAllocation(ctx context.Context, userID, id uint) (*models.TabdealGitAllocation, error) {
	allocation, err := s.tabdealAllocationRepo.ReturnByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if allocation.UserID != userID {
		return nil, gorm.ErrRecordNotFound
	}

	if allocation.Gift.Route == "" {
		route, err := s.tabdealAirdropRepo.FindRouteByPhrase(ctx, allocation.Gift.SearchPhrase)
		if err != nil {
			return nil, err
		}
		allocation.Gift.Route = route.Route
	}

	allocation, err = s.reSubmitAllocation(ctx, allocation)
	if err != nil {
		return nil, err
	}
	if allocation.Status == GiftStatusReceived {
		if err := s.refreshGiftStatus(ctx, allocation.GiftID); err != nil {
			return nil, err
		}
	}
	return allocation, nil
}

// refreshGiftStatus marks the gift received once every allocation of it is received.
func (s *Service) refreshGiftStatus(ctx context.Context, giftID uint) error {
	gift, err := s.tabdealAirdropRepo.GetGift(ctx, giftID)
	if err != nil {
		return err
	}
	for _, allocation := range gift.Allocations {
		if allocation.Status != GiftStatusReceived {
			return nil
		}
	}
	return s.tabdealAirdropRepo.UpdateGiftStatus(ctx, giftID, GiftStatusReceived, "")
}

func (s *Service) reSubmitAllocation(ctx context.Context, allocation *models.TabdealGitAllocation) (*models.TabdealGitAllocation, error) {
	user, err := s.UserRepo.GetByID(ctx, allocation.UserID)
	if err != nil {
		return nil, err
	}
	req := tabdeal.SubmitAirdropRequest{
		URL:             allocation.Gift.Route,
		SocialMediaCode: allocation.Gift.GiftCode,
		CurrencySymbol:  allocation.Currency,
		AuthToken:       user.TabdealEncryptedToken,
	}
	resp, err := s.tabdeal.SubmitAirdrop(ctx, req)
	if err != nil {
		allocation.ErrorMessage = err.Error()
		allocation.Status = GiftStatusFailed
	} else {
		allocation.Status = GiftStatusReceived
		allocation.ErrorMessage = ""
		if resp.Currency != "" {
			allocation.Currency = resp.Currency
		}
	}

	return allocation, s.tabdealAllocationRepo.Update(ctx, allocation)
}

// RetryCollectGift retries the failed allocations of every user, but returns the gift with only userID's allocations.
func (s *Service) RetryCollectGift(ctx context.Context, userID, id uint) (*models.TabdealAirdropGift, error) {
	gift, err := s.tabdealAirdropRepo.GetGift(ctx, id)
	if err != nil {
		return nil, err
	}

	if gift.Route == "" {
		route, err := s.tabdealAirdropRepo.FindRouteByPhrase(ctx, gift.SearchPhrase)
		if err != nil {
			return nil, err
		}
		gift.Route = route.Route
	}

	if len(gift.Allocations) == 0 {
		if err := s.resolveGiftCode(ctx, gift); err != nil {
			gift.Status = GiftStatusFailed
			gift.ErrorMessage = err.Error()
		} else if completelyCoolected, err := s.collectGift(ctx, gift); err != nil {
			gift.Status = GiftStatusFailed
			gift.ErrorMessage = err.Error()
		} else if completelyCoolected {
			gift.Status = GiftStatusReceived
		} else {
			gift.Status = GiftStatusPartialyReceived
		}
		if err := s.tabdealAirdropRepo.UpdateGift(ctx, gift); err != nil {
			return nil, err
		}
		return s.tabdealAirdropRepo.GetUserGift(ctx, userID, id)
	}

	collectedCount := 0
	for _, allocation := range gift.Allocations {
		if allocation.Status == GiftStatusFailed {
			allocation.Gift = gift
			if _, err := s.reSubmitAllocation(ctx, allocation); err != nil {
				continue
			}
		}
		if allocation.Status == GiftStatusReceived {
			collectedCount++
		}
	}
	status, errorMessage := GiftStatusPartialyReceived, gift.ErrorMessage
	if collectedCount == len(gift.Allocations) {
		status, errorMessage = GiftStatusReceived, ""
	}
	if err := s.tabdealAirdropRepo.UpdateGiftStatus(ctx, gift.ID, status, errorMessage); err != nil {
		return nil, err
	}

	return s.tabdealAirdropRepo.GetUserGift(ctx, userID, id)
}

// ProcessTelegramMessage handles both message patterns: the 4-step one carries the gift code
// in the message text, the 6-step one links to a tabdeal page that holds the code.
func (s *Service) ProcessTelegramMessage(ctx context.Context, chatID, messageID int64, raw string, links []string) (*models.TabdealAirdropGift, error) {
	phrases, code, pageURL, err := parseMessageCandidates(raw, links)
	phrase := ""
	if len(phrases) > 0 {
		phrase = phrases[0]
	}
	gift := &models.TabdealAirdropGift{
		TelegramMessageID: messageID,
		TelegramChatID:    chatID,
		RawMessage:        raw,
		SearchPhrase:      phrase,
		GiftCode:          code,
		GiftPageURL:       pageURL,
	}
	if err == nil {
		err = s.resolveGiftCode(ctx, gift)
	}
	if err != nil {
		gift.Status = GiftStatusFailed
		gift.ErrorMessage = err.Error()
		_ = s.tabdealAirdropRepo.CreateGift(ctx, gift)
		return gift, err
	}

	var route *models.TabdealAirdrop
	for _, candidate := range phrases {
		route, err = s.tabdealAirdropRepo.FindRouteByPhrase(ctx, candidate)
		if err == nil {
			gift.SearchPhrase = candidate
			break
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			gift.Status = GiftStatusRouteNotFound
			gift.ErrorMessage = err.Error()
			_ = s.tabdealAirdropRepo.CreateGift(ctx, gift)
			return gift, err
		}
	}
	if route == nil {
		gift.Status = GiftStatusRouteNotFound
		gift.ErrorMessage = gorm.ErrRecordNotFound.Error()
		_ = s.tabdealAirdropRepo.CreateGift(ctx, gift)
		return gift, gorm.ErrRecordNotFound
	}
	gift.Route = route.Route
	gift.Status = GiftStatusInitiated
	if err := s.tabdealAirdropRepo.CreateGift(ctx, gift); err != nil {
		return gift, err
	}

	if completelyCoolected, err := s.collectGift(ctx, gift); err != nil {
		gift.Status = GiftStatusFailed
		gift.ErrorMessage = err.Error()
	} else if completelyCoolected {
		gift.Status = GiftStatusReceived
	} else {
		gift.Status = GiftStatusPartialyReceived
	}
	return gift, s.tabdealAirdropRepo.UpdateGift(ctx, gift)
}

func (s *Service) collectGift(ctx context.Context, gift *models.TabdealAirdropGift) (bool, error) {
	users, err := s.UserRepo.List(ctx, map[string]any{
		"tabdeal_token_status = ?": consts.TabdealStatusEnable,
	})
	if err != nil {
		return false, err
	}
	if len(users) == 0 {
		return false, fmt.Errorf("no active user found")
	}

	collectCount := 0
	for _, user := range users {
		allocation := models.TabdealGitAllocation{
			GiftID:   gift.ID,
			Currency: selectGiftCurrency(),
			UserID:   user.ID,
		}

		req := tabdeal.SubmitAirdropRequest{
			URL:             gift.Route,
			SocialMediaCode: gift.GiftCode,
			CurrencySymbol:  allocation.Currency,
			AuthToken:       user.TabdealEncryptedToken,
		}
		if user.TabdealEncryptedToken == "" {
			allocation.ErrorMessage = "tabdeal token not found"
			allocation.Status = GiftStatusFailed
		} else {
			_, err := s.tabdeal.SubmitAirdrop(ctx, req)
			if err != nil {
				allocation.ErrorMessage = err.Error()
				allocation.Status = GiftStatusFailed
			} else {
				allocation.Status = GiftStatusReceived
				allocation.ErrorMessage = ""
				collectCount++
			}
		}
		_ = s.tabdealAllocationRepo.Create(ctx, &allocation)

	}

	if collectCount == len(users) {
		return true, nil
	}
	return false, nil
}

// resolveGiftCode fetches the gift code from the gift page when the message didn't carry it.
func (s *Service) resolveGiftCode(ctx context.Context, gift *models.TabdealAirdropGift) error {
	if gift.GiftCode != "" || gift.GiftPageURL == "" {
		return nil
	}
	code, err := s.tabdeal.FetchGiftCode(ctx, gift.GiftPageURL)
	if err != nil {
		return fmt.Errorf("fetch gift code from %s: %w", gift.GiftPageURL, err)
	}
	gift.GiftCode = code
	return nil
}

var (
	// search phrases sit between "جستجو کنید:" and the next step's keycap number (2️⃣ in the
	// 4-step pattern, 4️⃣/5️⃣ in the 6-step one).
	reTerms    = regexp.MustCompile(`(?s)جستجو کنید:\s*(.*?)\s*[0-9]\x{FE0F}?\x{20E3}`)
	reCode     = regexp.MustCompile(`(?s)کد زیر را وارد کنید:\s*(\S+)`)
	reGiftStep = regexp.MustCompile(`(?s)1\x{FE0F}?\x{20E3}(.*?)2\x{FE0F}?\x{20E3}`)
	reURL      = regexp.MustCompile(`https?://[^\s()<>]+`)
)

// parseMessageCandidates returns the search phrases plus either the gift code (4-step pattern)
// or the gift page URL to fetch it from (6-step pattern).
func parseMessageCandidates(raw string, links []string) ([]string, string, string, error) {
	phrases := []string{}
	matchesTerms := reTerms.FindStringSubmatch(raw)

	if len(matchesTerms) > 1 {
		lines := strings.Split(matchesTerms[1], "\n")
		for _, line := range lines {
			term := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "📌"))
			if term != "" {
				phrases = append(phrases, term)
			}
		}
	}

	matchesCode := reCode.FindStringSubmatch(raw)
	if len(matchesCode) > 1 {
		return phrases, strings.TrimSpace(matchesCode[1]), "", nil
	}

	// links from message entities come first since step-1 links are usually hidden behind text;
	// plain URLs inside the step-1 text are the fallback.
	candidates := append([]string{}, links...)
	if matchesStep := reGiftStep.FindStringSubmatch(raw); len(matchesStep) > 1 {
		candidates = append(candidates, reURL.FindAllString(matchesStep[1], -1)...)
	}
	for _, candidate := range candidates {
		if isTabdealURL(candidate) {
			return phrases, "", candidate, nil
		}
	}

	return phrases, "", "", nil
}

func isTabdealURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "tabdeal.org" || strings.HasSuffix(host, ".tabdeal.org")
}

func selectGiftCurrency() string {
	cryptos := []struct {
		Name   string
		Weight int
	}{
		{"BTC", 60},
		{"TRX", 15},
		{"XRP", 12},
		{"SHIB", 6},
		{"FLOKI", 4},
		{"NOT", 2},
		{"BABYDOGE", 1},
	}

	n := rand.IntN(100)

	for _, crypto := range cryptos {
		if n < crypto.Weight {
			return crypto.Name
		}
		n -= crypto.Weight
	}

	return "BTC"
}
