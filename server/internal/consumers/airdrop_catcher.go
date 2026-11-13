package consumers

import (
	"context"
	"fmt"
	"log"
	"time"
	"unicode/utf16"

	"math/rand/v2"

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
	var formatted *client.FormattedText
	switch message.Content.MessageContentType() {
	case client.TypeMessagePhoto:
		formatted = message.Content.(*client.MessagePhoto).Caption
	case client.TypeMessageText:
		formatted = message.Content.(*client.MessageText).Text
	}
	text, links := "", []string(nil)
	if formatted != nil {
		text, links = formatted.Text, extractLinks(formatted)
	}
	delay := time.Minute * time.Duration(rand.Int64N(16)+1)
	time.Sleep(delay)

	if _, err := c.service.ProcessTelegramMessage(ctx, message.ChatId, message.Id, text, links); err != nil {
		log.Printf("failed to process telegram airdrop message: %v", err)
	}
}

// extractLinks returns the message links in order: hidden text links and plain URLs.
func extractLinks(formatted *client.FormattedText) []string {
	var units []uint16
	links := []string{}
	for _, entity := range formatted.Entities {
		switch t := entity.Type.(type) {
		case *client.TextEntityTypeTextUrl:
			links = append(links, t.Url)
		case *client.TextEntityTypeUrl:
			// entity offsets are in UTF-16 code units
			if units == nil {
				units = utf16.Encode([]rune(formatted.Text))
			}
			start, end := int(entity.Offset), int(entity.Offset+entity.Length)
			if start >= 0 && end <= len(units) && start < end {
				links = append(links, string(utf16.Decode(units[start:end])))
			}
		}
	}
	return links
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
