package airdrop

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
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
func (s *Service) ListGifts(ctx context.Context, pg paginator.Params) ([]models.TabdealAirdropGift, int64, error) {
	return s.tabdealAirdropRepo.ListGifts(ctx, pg)
}
func (s *Service) GetGift(ctx context.Context, id uint) (*models.TabdealAirdropGift, error) {
	return s.tabdealAirdropRepo.GetGift(ctx, id)
}

func (s *Service) RetryCollectGiftAllocation(ctx context.Context, id uint) (*models.TabdealGitAllocation, error) {
	allocation, err := s.tabdealAllocationRepo.ReturnByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if allocation.Gift.Route == "" {
		route, err := s.tabdealAirdropRepo.FindRouteByPhrase(ctx, allocation.Gift.SearchPhrase)
		if err != nil {
			return nil, err
		}
		allocation.Gift.Route = route.Route
	}

	return s.reSubmitAllocation(ctx, allocation)
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

func (s *Service) RetryCollectGift(ctx context.Context, id uint) (*models.TabdealAirdropGift, error) {
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

	collectedCount := 0
	for i, allocation := range gift.Allocations {
		if allocation.Status == GiftStatusFailed {
			alloc, err := s.reSubmitAllocation(ctx, allocation)
			if err == nil {
				gift.Allocations[i] = alloc
				collectedCount++
			}
		} else {
			collectedCount++
		}
	}
	if collectedCount == len(gift.Allocations) {
		gift.Status = GiftStatusReceived
		gift.ErrorMessage = ""
		return gift, s.tabdealAirdropRepo.UpdateGift(ctx, gift)
	}

	return gift, nil
}

func (s *Service) ProcessTelegramMessage(ctx context.Context, chatID, messageID int64, raw string) (*models.TabdealAirdropGift, error) {
	phrases, code, err := parseMessageCandidates(raw)
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

func parseMessageCandidates(raw string) ([]string, string, error) {
	phrases := []string{}
	reTerms := regexp.MustCompile(`(?s)جستجو کنید:\s*(?:📌)?\s*(.*?)\s*2️⃣`)
	matchesTerms := reTerms.FindStringSubmatch(raw)

	if len(matchesTerms) > 1 {
		lines := strings.Split(matchesTerms[1], "\n")
		for _, line := range lines {
			term := strings.TrimSpace(line)
			if term != "" {
				phrases = append(phrases, term)
			}
		}
	}

	reCode := regexp.MustCompile(`(?s)کد زیر را وارد کنید:\s*(\S+)`)
	matchesCode := reCode.FindStringSubmatch(raw)

	code := ""
	if len(matchesCode) > 1 {
		code = strings.TrimSpace(matchesCode[1])
	}

	return phrases, code, nil
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
