package tgaccount

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/h-varmazyar/gatec/configs"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/h-varmazyar/gatec/pkg/paginator"
	"github.com/h-varmazyar/gatec/pkg/tgclient"
	"github.com/sirupsen/logrus"
	"github.com/zelenin/go-tdlib/client"
)

// stateWaitTimeout is how long we wait for TDLib to transition states
// after we submit credentials. This covers network latency + Telegram processing.
const stateWaitTimeout = 30 * time.Second

// Service provides telegram account management over REST APIs.
// It handles the multi-step authentication flow by coordinating
// between the synchronous HTTP handlers and the async TDLib client.
type Service struct {
	manager       *tgclient.Manager
	accountRepo   *repositories.TelegramAccountRepository
	filterRepo    *repositories.TelegramSubscriptionFilterRepository
	proxyRepo     *repositories.ProxyRepository
	subscriptions []Subscription
	// onSessionReadyHook is called when a session becomes ready (used by workers for proxy assignment)
	onSessionReadyHook func(ctx context.Context, sessionID string)
}

// Subscription represents a consumer listening for telegram updates.
type Subscription struct {
	Consumer string
	Filter   UpdateFilter
	Updates  chan<- client.Update
}

// UpdateFilter defines which updates a subscriber wants to receive.
type UpdateFilter struct {
	ChatIDs     []int64
	UpdateTypes []string
	AccountIDs  []string
}

// NewService creates a new tgaccount service.
func NewService(
	cfg *configs.TgAccountService,
	// manager *tgclient.Manager,
	accountRepo *repositories.TelegramAccountRepository,
	filterRepo *repositories.TelegramSubscriptionFilterRepository,
	proxyRepo *repositories.ProxyRepository,
	// databaseBaseDir string,
) *Service {
	return &Service{
		manager:     tgclient.NewManager(cfg.AuthManager),
		accountRepo: accountRepo,
		filterRepo:  filterRepo,
		proxyRepo:   proxyRepo,
		// databaseBaseDir: databaseBaseDir,
	}
}

// GetManager returns the underlying tgclient.Manager instance.
// This is used by workers that need to interact with TDLib sessions directly.
func (s *Service) GetManager() *tgclient.Manager {
	return s.manager
}

// SetOnSessionReadyHook registers a callback that is called when a loaded session becomes ready.
// This is used by the connection monitor worker to assign proxies to sessions.
func (s *Service) SetOnSessionReadyHook(hook func(ctx context.Context, sessionID string)) {
	s.onSessionReadyHook = hook
}

// --- Authentication Flow ---

// CreateAccountResponse is returned when a new account session is started.
type CreateAccountResponse struct {
	SessionID string `json:"session_id"`
	State     string `json:"state"`
	Error     string `json:"error,omitempty"`
}

// CreateAccount starts a new telegram account authentication session.
// It returns a session ID and the first state the client needs to respond to.
func (s *Service) CreateAccount(ctx context.Context, sessionType string) (*CreateAccountResponse, error) {
	var st tgclient.SessionType
	switch sessionType {
	case "client":
		st = tgclient.SessionTypeClient
	case "bot":
		st = tgclient.SessionTypeBot
	default:
		return nil, fmt.Errorf("invalid session type: %s (must be 'client' or 'bot')", sessionType)
	}

	proxy, err := s.proxyRepo.GetByID(ctx, 1)
	if err != nil {
		return nil, err
	}

	proxyConfig := tgclient.ProxyConfig{
		Server:   proxy.Server,
		Port:     proxy.Port,
		Secret:   proxy.Secret,
		Username: proxy.Username,
		Password: proxy.Password,
		Type:     tgclient.ProxyType(proxy.Type),
	}

	sessionID := uuid.New().String()

	_, err = s.manager.CreateSession(ctx, sessionID, st, &proxyConfig)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	// Start a background goroutine that cleans up if auth is not completed in time
	go s.watchAuthTimeout(sessionID)

	// Wait for TDLib to tell us what it needs first (phone number usually)
	state, err := s.manager.WaitForState(ctx, sessionID, stateWaitTimeout)
	if err != nil {
		s.cleanupFailedSession(sessionID)
		return nil, fmt.Errorf("wait for initial state: %w", err)
	}

	return &CreateAccountResponse{
		SessionID: sessionID,
		State:     string(state),
	}, nil
}

// watchAuthTimeout waits for the auth timeout duration. If the session is still
// not ready by then, it cleans up the database files.
func (s *Service) watchAuthTimeout(sessionID string) {
	timeout := s.manager.GetAuthTimeout()
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	<-timer.C

	state, err := s.manager.GetState(sessionID)
	if err != nil {
		// Session already removed, nothing to do
		return
	}

	// Only cleanup if the session never reached Ready state
	if state == tgclient.SessionStateReady {
		return
	}

	logrus.WithField("session_id", sessionID).Warn("auth timeout reached, cleaning up session")
	s.cleanupFailedSession(sessionID)
}

// SubmitPhoneResponse is returned after submitting a phone number.
type SubmitPhoneResponse struct {
	SessionID string `json:"session_id"`
	State     string `json:"state"`
}

// SubmitPhone submits a phone number and waits for the next state transition.
func (s *Service) SubmitPhone(ctx context.Context, sessionID, phone string) (*SubmitPhoneResponse, error) {
	if err := s.manager.SubmitPhone(sessionID, phone); err != nil {
		return nil, err
	}

	// Wait for TDLib to process the phone and tell us what's next (code, error, etc.)
	state, err := s.manager.WaitForState(ctx, sessionID, stateWaitTimeout)
	if err != nil {
		s.cleanupFailedSession(sessionID)
		return nil, fmt.Errorf("waiting for state after phone submission: %w", err)
	}

	if state == tgclient.SessionStateError {
		s.cleanupFailedSession(sessionID)
		return nil, fmt.Errorf("authentication failed after phone submission")
	}

	return &SubmitPhoneResponse{
		SessionID: sessionID,
		State:     string(state),
	}, nil
}

// SubmitCodeResponse is returned after submitting a verification code.
type SubmitCodeResponse struct {
	SessionID string `json:"session_id"`
	State     string `json:"state"`
}

// SubmitCode submits the OTP code and waits for the next state transition.
func (s *Service) SubmitCode(ctx context.Context, sessionID, code string) (*SubmitCodeResponse, error) {
	if err := s.manager.SubmitCode(sessionID, code); err != nil {
		return nil, err
	}

	state, err := s.manager.WaitForState(ctx, sessionID, stateWaitTimeout)
	if err != nil {
		s.cleanupFailedSession(sessionID)
		return nil, fmt.Errorf("waiting for state after code submission: %w", err)
	}

	if state == tgclient.SessionStateError {
		s.cleanupFailedSession(sessionID)
		return nil, fmt.Errorf("authentication failed after code submission")
	}

	// If we reached ready state, persist the account
	if state == tgclient.SessionStateReady {
		s.onSessionReady(ctx, sessionID)
	}

	return &SubmitCodeResponse{
		SessionID: sessionID,
		State:     string(state),
	}, nil
}

// SubmitPasswordResponse is returned after submitting a 2FA password.
type SubmitPasswordResponse struct {
	SessionID string `json:"session_id"`
	State     string `json:"state"`
}

// SubmitPassword submits the 2FA password and waits for the next state transition.
func (s *Service) SubmitPassword(ctx context.Context, sessionID, password string) (*SubmitPasswordResponse, error) {
	if err := s.manager.SubmitPassword(sessionID, password); err != nil {
		return nil, err
	}

	state, err := s.manager.WaitForState(ctx, sessionID, stateWaitTimeout)
	if err != nil {
		s.cleanupFailedSession(sessionID)
		return nil, fmt.Errorf("waiting for state after password submission: %w", err)
	}

	if state == tgclient.SessionStateError {
		s.cleanupFailedSession(sessionID)
		return nil, fmt.Errorf("authentication failed after password submission")
	}

	if state == tgclient.SessionStateReady {
		s.onSessionReady(ctx, sessionID)
	}

	return &SubmitPasswordResponse{
		SessionID: sessionID,
		State:     string(state),
	}, nil
}

// GetSessionState returns the current state of a session.
func (s *Service) GetSessionState(ctx context.Context, sessionID string) (string, error) {
	state, err := s.manager.GetState(sessionID)
	if err != nil {
		return "", err
	}
	return string(state), nil
}

// --- Re-authentication Flow ---

// ReAuthenticateResponse is returned when starting a re-authentication flow.
type ReAuthenticateResponse struct {
	SessionID string `json:"session_id"`
	State     string `json:"state"`
}

// ReAuthenticate starts the re-authentication process for an existing account.
// It creates a new TDLib session using the account's stored phone number,
// submits the phone automatically, and returns the next state (usually wait_code).
// After this, the client uses the existing SubmitCode/SubmitPassword endpoints
// with the same session_id to complete the flow.
func (s *Service) ReAuthenticate(ctx context.Context, accountID string) (*ReAuthenticateResponse, error) {
	// Fetch the account from DB
	account, err := s.accountRepo.Get(ctx, map[string]any{"id = ?": accountID})
	if err != nil {
		return nil, fmt.Errorf("account not found: %w", err)
	}

	if account.Mobile == nil || *account.Mobile == "" {
		return nil, fmt.Errorf("account has no phone number stored, cannot re-authenticate")
	}

	sessionID := accountID

	// If there's already an active session for this account, close it first
	if _, err := s.manager.GetSession(sessionID); err == nil {
		_ = s.manager.CloseSession(sessionID)
	}

	// Update status to needs_login
	_ = s.accountRepo.UpdateStatus(ctx, sessionID, models.TelegramAccountStatusNeedsLogin)

	proxy, err := s.proxyRepo.GetByID(ctx, 1)
	if err != nil {
		return nil, err
	}

	proxyConfig := &tgclient.ProxyConfig{
		Server:   proxy.Server,
		Port:     proxy.Port,
		Secret:   proxy.Secret,
		Username: proxy.Username,
		Password: proxy.Password,
	}

	// Create a new TDLib session
	_, err = s.manager.CreateSession(ctx, sessionID, tgclient.SessionTypeClient, proxyConfig)
	if err != nil {
		return nil, fmt.Errorf("create session for re-auth: %w", err)
	}

	// Start a background goroutine that cleans up if auth is not completed in time
	go s.watchAuthTimeout(sessionID)

	// Wait for TDLib to ask for phone number
	state, err := s.manager.WaitForState(ctx, sessionID, stateWaitTimeout)
	if err != nil {
		s.cleanupFailedSession(sessionID)
		return nil, fmt.Errorf("wait for initial state: %w", err)
	}

	// If TDLib is asking for phone, submit it automatically
	if state == tgclient.SessionStateWaitPhone {
		if err := s.manager.SubmitPhone(sessionID, *account.Mobile); err != nil {
			s.cleanupFailedSession(sessionID)
			return nil, fmt.Errorf("submit phone for re-auth: %w", err)
		}

		// Wait for next state after phone submission
		state, err = s.manager.WaitForState(ctx, sessionID, stateWaitTimeout)
		if err != nil {
			s.cleanupFailedSession(sessionID)
			return nil, fmt.Errorf("waiting for state after phone submission: %w", err)
		}

		if state == tgclient.SessionStateError {
			s.cleanupFailedSession(sessionID)
			return nil, fmt.Errorf("authentication failed during re-auth phone submission")
		}
	}

	// If already ready (session was still valid), update DB and start listening
	if state == tgclient.SessionStateReady {
		s.onSessionReady(ctx, sessionID)
	}

	return &ReAuthenticateResponse{
		SessionID: sessionID,
		State:     string(state),
	}, nil
}

// --- Account persistence ---

func (s *Service) onSessionReady(ctx context.Context, sessionID string) {
	log := logrus.WithField("session_id", sessionID)
	log.Info("session ready, fetching user info and persisting account")

	// Fetch user info from Telegram
	tdClient, err := s.manager.GetClient(sessionID)
	if err != nil {
		log.WithError(err).Error("failed to get TDLib client for user info")
		return
	}

	me, err := tdClient.GetMe()
	if err != nil {
		log.WithError(err).Error("failed to call GetMe()")
		return
	}

	// Extract username
	username := ""
	if me.Usernames != nil && len(me.Usernames.ActiveUsernames) > 0 {
		username = me.Usernames.ActiveUsernames[0]
	}

	// Build account name from first + last name
	accountName := me.FirstName
	if me.LastName != "" {
		accountName += " " + me.LastName
	}

	// Build phone
	mobile := me.PhoneNumber

	dbDir := s.manager.GetDatabaseDir(sessionID)
	account := &models.TelegramAccount{
		ID:                uuid.MustParse(sessionID),
		Role:              "client",
		Status:            models.TelegramAccountStatusReady,
		DatabaseDirectory: filepath.Join(dbDir, "database"),
		FilesDirectory:    filepath.Join(dbDir, "files"),
		Username:          username,
		AccountName:       accountName,
		UserID:            me.Id,
	}
	if mobile != "" {
		account.Mobile = &mobile
	}

	if err := s.accountRepo.Upsert(ctx, account); err != nil {
		log.WithError(err).Error("failed to persist account")
		return
	}

	log.WithFields(logrus.Fields{
		"username":     username,
		"account_name": accountName,
		"user_id":      me.Id,
	}).Info("account persisted successfully")

	// Start listening for updates
	go s.listenForUpdates(ctx, sessionID)
}

// cleanupFailedSession removes the TDLib database files for a session that
// failed to authenticate. Only sessions that completed auth should keep their files.
func (s *Service) cleanupFailedSession(sessionID string) {
	log := logrus.WithField("session_id", sessionID)

	dbDir := s.manager.GetDatabaseDir(sessionID)
	if err := os.RemoveAll(dbDir); err != nil {
		log.WithError(err).Error("failed to cleanup database directory")
	} else {
		log.Info("cleaned up database directory for failed session")
	}

	// Also remove the session from the manager
	_ = s.manager.CloseSession(sessionID)
}

// --- Update dispatching ---

// Subscribe registers a consumer for telegram updates.
func (s *Service) Subscribe(consumer string, filter UpdateFilter, updates chan<- client.Update) {
	s.subscriptions = append(s.subscriptions, Subscription{
		Consumer: consumer,
		Filter:   filter,
		Updates:  updates,
	})
}

func (s *Service) listenForUpdates(ctx context.Context, sessionID string) {
	log := logrus.WithField("session_id", sessionID)

	listener, err := s.manager.GetListener(sessionID)
	if err != nil {
		log.WithError(err).Error("failed to get listener")
		return
	}

	log.Info("listening for updates")

	for {
		select {
		case <-ctx.Done():
			return
		case typ, ok := <-listener.Updates:
			if !ok {
				log.Info("listener channel closed")
				return
			}
			update, isUpdate := typ.(client.Update)
			if !isUpdate {
				continue
			}
			// switch u := update.(type) {
			// case *client.UpdateNewMessage:
			// 	fmt.Println(u.Message)
			// default:
			// 	fmt.Println(u.UpdateType())
			// }
			s.dispatch(sessionID, update)
		}
	}
}

func (s *Service) dispatch(sessionID string, update client.Update) {
	for _, sub := range s.subscriptions {
		if !sub.Filter.matches(sessionID, update) {
			continue
		}
		select {
		case sub.Updates <- update:
		default:
			logrus.Warnf("subscriber %s channel full, dropping update", sub.Consumer)
		}
	}
}

func (f UpdateFilter) matches(accountID string, update client.Update) bool {
	if len(f.AccountIDs) > 0 && !containsStr(f.AccountIDs, accountID) {
		return false
	}
	if len(f.UpdateTypes) > 0 && !containsStr(f.UpdateTypes, update.UpdateType()) {
		return false
	}
	if len(f.ChatIDs) > 0 {
		chatID, ok := extractChatID(update)
		if !ok || !containsInt64(f.ChatIDs, chatID) {
			return false
		}
	}
	return true
}

// --- Account loading from DB ---

// LoadAccounts loads previously authenticated accounts from the database
// and re-establishes their TDLib sessions.
func (s *Service) LoadAccounts(ctx context.Context) error {
	accounts, err := s.accountRepo.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("list accounts: %w", err)
	}

	log := logrus.WithField("count", len(accounts))
	log.Info("loading accounts from database")

	proxy, err := s.proxyRepo.GetByID(ctx, 1)
	if err != nil {
		return err
	}

	proxyConfig := &tgclient.ProxyConfig{
		Server:   proxy.Server,
		Port:     proxy.Port,
		Secret:   proxy.Secret,
		Username: proxy.Username,
		Password: proxy.Password,
		Type:     tgclient.ProxyType(proxy.Type),
	}
	log.WithField("id", proxy.ID).Info("proxy")
	for _, acc := range accounts {
		if acc.Status != models.TelegramAccountStatusReady {
			logrus.Infof("skipping account %s (status: %s)", acc.ID, acc.Status)
			continue
		}

		sessionID := acc.ID.String()
		_, err = s.manager.CreateSession(ctx, sessionID, tgclient.SessionTypeClient, proxyConfig)
		if err != nil {
			logrus.WithError(err).Errorf("failed to load account %s", sessionID)
			_ = s.accountRepo.UpdateStatus(ctx, sessionID, models.TelegramAccountStatusDisconnected)
			continue
		}

		// Wait to see if it reconnects (TDLib uses stored session)
		go s.handleLoadedSession(ctx, sessionID, proxyConfig)
	}

	return nil
}

func (s *Service) handleLoadedSession(ctx context.Context, sessionID string, proxyConfig *tgclient.ProxyConfig) {
	log := logrus.WithField("session_id", sessionID)

	// Give TDLib more time to restore a session from disk (includes DB init)
	state, err := s.manager.WaitForState(ctx, sessionID, 60*time.Second)
	if err != nil {
		log.WithError(err).Error("timeout waiting for loaded session state")
		_ = s.accountRepo.UpdateStatus(ctx, sessionID, models.TelegramAccountStatusDisconnected)
		return
	}

	switch state {
	case tgclient.SessionStateReady:
		log.Info("loaded session is ready")
		_ = s.accountRepo.UpdateStatus(ctx, sessionID, models.TelegramAccountStatusReady)

		// Apply proxy if there's an OnSessionReady callback registered
		if s.onSessionReadyHook != nil {
			s.onSessionReadyHook(ctx, sessionID)
		}

		go s.listenForUpdates(ctx, sessionID)
	case tgclient.SessionStateWaitPhone, tgclient.SessionStateWaitCode, tgclient.SessionStateWaitPassword:
		// TDLib couldn't restore the session — it's asking for credentials again.
		// This means the session was invalidated (revoked, expired, etc.)
		log.Warnf("loaded session needs re-auth (state: %s) — session may have been invalidated by Telegram", state)
		_ = s.accountRepo.UpdateStatus(ctx, sessionID, models.TelegramAccountStatusNeedsLogin)
		// Close the session since we can't do anything with it during startup
		_ = s.manager.CloseSession(sessionID)
	default:
		log.Infof("loaded session in unexpected state: %s", state)
		_ = s.accountRepo.UpdateStatus(ctx, sessionID, models.TelegramAccountStatusDisconnected)
		_ = s.manager.CloseSession(sessionID)
	}
}

// --- List/Filter operations ---

// ListAccounts returns a paginated list of accounts from the database.
func (s *Service) ListAccounts(ctx context.Context, pg paginator.Params) ([]models.TelegramAccount, int64, error) {
	return s.accountRepo.List(ctx, pg, nil)
}

// CreateSubscriptionFilter creates a new subscription filter.
func (s *Service) CreateSubscriptionFilter(ctx context.Context, accountID string, chatID int64) (*models.TelegramSubscriptionFilter, error) {
	if accountID == "" {
		return nil, fmt.Errorf("account_id is required")
	}
	if chatID == 0 {
		return nil, fmt.Errorf("chat_id is required")
	}

	filter := &models.TelegramSubscriptionFilter{
		AccountID: accountID,
		ChatID:    chatID,
		IsActive:  true,
	}

	if err := s.filterRepo.Create(ctx, filter); err != nil {
		return nil, err
	}
	return filter, nil
}

// ListSubscriptionFilters returns a paginated list of subscription filters.
func (s *Service) ListSubscriptionFilters(ctx context.Context, pg paginator.Params) ([]models.TelegramSubscriptionFilter, int64, error) {
	return s.filterRepo.List(ctx, pg)
}

// ListActiveSubscriptionFilters returns all active subscription filters.
func (s *Service) ListActiveSubscriptionFilters(ctx context.Context) ([]models.TelegramSubscriptionFilter, error) {
	return s.filterRepo.ListActive(ctx)
}

// DeleteSubscriptionFilter deletes a subscription filter by ID.
func (s *Service) DeleteSubscriptionFilter(ctx context.Context, id uint) error {
	return s.filterRepo.Delete(ctx, id)
}

// --- Helpers ---

func extractChatID(update client.Update) (int64, bool) {
	switch u := update.(type) {
	case *client.UpdateNewMessage:
		if u.Message != nil {
			return u.Message.ChatId, true
		}
	case *client.UpdateMessageContent:
		return u.ChatId, true
	case *client.UpdateMessageEdited:
		return u.ChatId, true
	case *client.UpdateMessageSendSucceeded:
		if u.Message != nil {
			return u.Message.ChatId, true
		}
	case *client.UpdateMessageSendFailed:
		if u.Message != nil {
			return u.Message.ChatId, true
		}
	}
	return 0, false
}

func containsStr(slice []string, target string) bool {
	for _, v := range slice {
		if v == target {
			return true
		}
	}
	return false
}

func containsInt64(slice []int64, target int64) bool {
	for _, v := range slice {
		if v == target {
			return true
		}
	}
	return false
}
