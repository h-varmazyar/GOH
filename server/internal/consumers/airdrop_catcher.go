package consumers

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"time"

	"github.com/h-varmazyar/gatec/internal/services/airdrop"
	"github.com/h-varmazyar/gatec/internal/services/tgaccount"
	"github.com/zelenin/go-tdlib/client"
)

type AirdropCatcher struct {
	telegramService *tgaccount.Service
	service         *airdrop.Service
	tgUpdateChan    chan client.Update
	filter          tgaccount.UpdateFilter
}

func NewAirdropCatcher(telegramService *tgaccount.Service, service *airdrop.Service) *AirdropCatcher {
	return &AirdropCatcher{telegramService: telegramService, service: service}
}

func (c *AirdropCatcher) Start(ctx context.Context) {
	if err := c.loadFiltersFromDB(ctx); err != nil {
		log.Printf("failed to load subscription filters from DB: %v", err)
		return
	}

	if len(c.filter.ChatIDs) == 0 {
		log.Println("no active subscription filters found, airdrop catcher not started")
		return
	}

	fmt.Printf("subscribing for filter: %+v\n", c.filter)

	c.tgUpdateChan = make(chan client.Update, 1000)
	c.telegramService.Subscribe("airdrop_catcher", c.filter, c.tgUpdateChan)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case update := <-c.tgUpdateChan:
				c.handleUpdate(ctx, update)
			}
		}
	}()

	log.Printf("airdrop catcher started with %d chat filters", len(c.filter.ChatIDs))
}

func (c *AirdropCatcher) loadFiltersFromDB(ctx context.Context) error {
	filters, err := c.telegramService.ListActiveSubscriptionFilters(ctx)
	if err != nil {
		return err
	}

	chatIDs := make([]int64, 0, len(filters))
	accountIDs := make([]string, 0, len(filters))
	for _, f := range filters {
		chatIDs = append(chatIDs, f.ChatID)
		accountIDs = append(accountIDs, f.AccountID)
	}

	c.filter = tgaccount.UpdateFilter{
		ChatIDs:    chatIDs,
		AccountIDs: accountIDs,
	}

	return nil
}

func (c *AirdropCatcher) handleUpdate(ctx context.Context, update client.Update) {
	switch u := update.(type) {
	case *client.UpdateNewMessage:
		println("message")
		if u.Message == nil {
			return
		}
		c.handleMessage(ctx, u.Message)
	case *client.UpdateChatLastMessage:
		println("last message")
		if u.LastMessage == nil {
			return
		}
		c.handleMessage(ctx, u.LastMessage)
	default:
		log.Printf("unhandled update type: %T", update)
	}
}

func (c *AirdropCatcher) handleMessage(ctx context.Context, message *client.Message) {
	text := ""
	switch message.Content.MessageContentType() {
	case client.TypeMessagePhoto:
		text = message.Content.(*client.MessagePhoto).Caption.Text
	case client.TypeMessageText:
		text = message.Content.(*client.MessageText).Text.Text
	}
	delay := time.Minute * time.Duration(rand.Int64N(16)+1)
	time.Sleep(delay)

	if _, err := c.service.ProcessTelegramMessage(ctx, message.ChatId, message.Id, text); err != nil {
		log.Printf("failed to process telegram airdrop message: %v", err)
	}
}

type TelegramAdminNotifier struct {
	telegramService *tgaccount.Service
	role            string
	chatID          int64
}

func NewTelegramAdminNotifier(telegramService *tgaccount.Service, role string, chatID int64) *TelegramAdminNotifier {
	return &TelegramAdminNotifier{telegramService: telegramService, role: role, chatID: chatID}
}

func (n *TelegramAdminNotifier) NotifyAdmin(ctx context.Context, text string) {
	// TODO: implement using tgaccount service's GetClient when needed
}
