package airdrop

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/h-varmazyar/gatec/configs"
	"github.com/h-varmazyar/gatec/internal/adapters/crypto/tabdeal"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/h-varmazyar/gatec/internal/services/tgaccount"
	"github.com/h-varmazyar/gatec/pkg/paginator"
	"gorm.io/gorm"
)

const (
	GiftStatusReceived      = "received"
	GiftStatusFailed        = "failed"
	GiftStatusRouteNotFound = "route_not_found"
)

var SupportedCurrencies = []string{"XRP", "SHIB", "ETH", "DOGE", "TRX", "USDT"}

type AdminNotifier interface {
	NotifyAdmin(ctx context.Context, text string)
}

type Service struct {
	repo             *repositories.TabdealAirdropRepository
	wallex           *tabdeal.Adapter
	tgAccountService *tgaccount.Service
}

func NewService(cfg configs.AirdropService, repo *repositories.TabdealAirdropRepository, wallexAdapter *tabdeal.Adapter, tgAccountService *tgaccount.Service) *Service {
	return &Service{
		repo:             repo,
		wallex:           wallexAdapter,
		tgAccountService: tgAccountService,
	}
}

func (s *Service) CreateRoute(ctx context.Context, phrase, route string) (*models.TabdealAirdrop, error) {
	item := &models.TabdealAirdrop{Phrase: strings.TrimSpace(phrase), Route: strings.TrimSpace(route)}
	if item.Phrase == "" || item.Route == "" {
		return nil, fmt.Errorf("phrase and route are required")
	}
	return item, s.repo.UpsertRoute(ctx, item)
}

func (s *Service) ListRoutes(ctx context.Context, pg paginator.Params) ([]models.TabdealAirdrop, int64, error) {
	return s.repo.ListRoutes(ctx, pg)
}
func (s *Service) ListGifts(ctx context.Context, pg paginator.Params) ([]models.TabdealAirdropGift, int64, error) {
	return s.repo.ListGifts(ctx, pg)
}
func (s *Service) GetGift(ctx context.Context, id uint) (*models.TabdealAirdropGift, error) {
	return s.repo.GetGift(ctx, id)
}

func (s *Service) RetryCollectGift(ctx context.Context, id uint) (*models.TabdealAirdropGift, error) {
	gift, err := s.repo.GetGift(ctx, id)
	if err != nil {
		return nil, err
	}

	if gift.Route == "" {
		route, err := s.repo.FindRouteByPhrase(ctx, gift.SearchPhrase)
		if err != nil {
			return nil, err
		}
		gift.Route = route.Route
	}

	req := tabdeal.SubmitAirdropRequest{
		URL:             gift.Route,
		SocialMediaCode: gift.GiftCode,
		CurrencySymbol:  gift.Currency,
	}
	resp, err := s.wallex.SubmitAirdrop(ctx, req)
	if err != nil {
		gift.ErrorMessage = err.Error()
		_ = s.repo.UpdateGift(ctx, gift)
		return gift, err
	}

	gift.Status = GiftStatusReceived
	gift.ErrorMessage = ""
	if resp.Currency != "" {
		gift.Currency = resp.Currency
	}
	return gift, s.repo.UpdateGift(ctx, gift)
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
		Currency:          SupportedCurrencies[0],
	}
	if err != nil {
		gift.Status = GiftStatusFailed
		gift.ErrorMessage = err.Error()
		_ = s.repo.CreateGift(ctx, gift)
		return gift, err
	}

	var route *models.TabdealAirdrop
	for _, candidate := range phrases {
		route, err = s.repo.FindRouteByPhrase(ctx, candidate)
		if err == nil {
			gift.SearchPhrase = candidate
			break
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			gift.Status = GiftStatusRouteNotFound
			gift.ErrorMessage = err.Error()
			_ = s.repo.CreateGift(ctx, gift)
			return gift, err
		}
	}
	if route == nil {
		gift.Status = GiftStatusRouteNotFound
		gift.ErrorMessage = gorm.ErrRecordNotFound.Error()
		_ = s.repo.CreateGift(ctx, gift)
		fmt.Println(s.notifyAdmin(fmt.Sprintf("airdrop search phrase not found: %s\ncode: %s", strings.Join(phrases, ", "), code)))
		return gift, gorm.ErrRecordNotFound
	}
	gift.Route = route.Route
	gift.Status = GiftStatusFailed
	if err := s.repo.CreateGift(ctx, gift); err != nil {
		return gift, err
	}

	req := tabdeal.SubmitAirdropRequest{
		URL:             route.Route,
		SocialMediaCode: code,
		CurrencySymbol:  gift.Currency,
	}
	resp, err := s.wallex.SubmitAirdrop(ctx, req)
	if err != nil {
		gift.ErrorMessage = err.Error()
		fmt.Println(s.notifyAdmin(fmt.Sprintf("wallex authorization failed while submitting airdrop\nphrase: %s\nroute: %s\ncode: %s\nerror: %s", phrase, route.Route, code, err.Error())))
		_ = s.repo.UpdateGift(ctx, gift)
		return gift, err
	}

	gift.Status = GiftStatusReceived
	gift.ErrorMessage = ""
	if resp.Currency != "" {
		gift.Currency = resp.Currency
	}
	return gift, s.repo.UpdateGift(ctx, gift)
}

func (s *Service) notifyAdmin(text string) error {
	return nil
	// accountManager := s.tgAccountService.GetManager()

	// tgClient, err := accountManager.GetClient("b02eef38-c89d-4227-a07a-2c12b9757307")
	// if err != nil {
	// 	return err
	// }

	// req := &client.SendMessageRequest{
	// 	ChatId:              1689926,
	// 	InputMessageContent: &client.InputMessageText{Text: &client.FormattedText{Text: text}},
	// }
	// _, err = tgClient.SendMessage(req)

	// return err
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
