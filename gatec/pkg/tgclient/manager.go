package tgclient

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zelenin/go-tdlib/client"
)

// Manager manages multiple telegram sessions.
// It provides a simple, synchronous API for creating sessions and submitting
// authentication credentials, while internally managing the async TDLib flow.
type Manager struct {
	cfg      *Config
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewManager creates a new session manager.
func NewManager(cfg *Config) *Manager {
	_, _ = client.SetLogVerbosityLevel(&client.SetLogVerbosityLevelRequest{
		NewVerbosityLevel: 1,
	})

	return &Manager{
		cfg:      cfg,
		sessions: make(map[string]*Session),
	}
}

// CreateSession creates a new session and starts the TDLib authorization flow in background.
// The returned Session will be in an initial state; callers should use WaitForState
// to determine what input is needed next (phone, code, or password).
func (m *Manager) CreateSession(ctx context.Context, sessionID string, sessionType SessionType, proxy *ProxyConfig) (*Session, error) {
	m.mu.Lock()
	if _, exists := m.sessions[sessionID]; exists {
		m.mu.Unlock()
		return nil, fmt.Errorf("session already exists: %s", sessionID)
	}
	m.mu.Unlock()

	if sessionType == SessionTypeBot {
		return nil, fmt.Errorf("bot sessions are not yet supported in the new flow")
	}

	dbDir := filepath.Join(m.cfg.DatabaseBaseDir, sessionID)
	tdlibParams := &client.SetTdlibParametersRequest{
		UseTestDc:           false,
		DatabaseDirectory:   filepath.Join(dbDir, "database"),
		FilesDirectory:      filepath.Join(dbDir, "files"),
		UseFileDatabase:     true,
		UseChatInfoDatabase: true,
		UseMessageDatabase:  true,
		UseSecretChats:      false,
		ApiId:               m.cfg.APIID,
		ApiHash:             m.cfg.APIHash,
		SystemLanguageCode:  "en",
		DeviceModel:         "Server",
		SystemVersion:       "1.0.0",
		ApplicationVersion:  "1.0.0",
	}

	authorizer := newClientAuthorizer(tdlibParams, m.cfg.AuthTimeout)

	session := &Session{
		id:          sessionID,
		sessionType: sessionType,
		state:       SessionStateWaitPhone, // initial assumed state
		authorizer:  authorizer,
		createdAt:   time.Now(),
	}

	if proxy != nil {
		session.proxy = proxy
	}

	m.mu.Lock()
	m.sessions[sessionID] = session
	m.mu.Unlock()

	// Start TDLib client in a goroutine. NewClient blocks until auth is complete or fails.
	go m.runAuth(session)

	return session, nil
}

// runAuth runs the blocking NewClient call. When it returns, the session is either
// Ready or in an error state.
func (m *Manager) runAuth(session *Session) {
	log := logrus.WithField("session_id", session.id)
	log.Info("starting TDLib authorization")

	var opts []client.Option
	if session.proxy != nil {
		parsedProxy := session.proxy.toAddProxyRequest()
		println("parsed proxy:", parsedProxy.Enable, parsedProxy.Port, parsedProxy.Type.ProxyTypeType())
		option := client.WithProxy(parsedProxy)
		opts = append(opts, option)
	}

	tdClient, err := client.NewClient(session.authorizer, opts...)
	if err != nil {
		log.WithError(err).Error("TDLib authorization failed")
		m.mu.Lock()
		session.state = SessionStateError
		session.stateErr = err.Error()
		m.mu.Unlock()

		// Notify via stateCh so WaitForState unblocks with error state
		select {
		case session.authorizer.stateCh <- SessionStateError:
		default:
		}

		// Notify via errCh so cleanup can be triggered externally
		select {
		case session.authorizer.errCh <- err:
		default:
		}
		return
	}

	log.Info("TDLib authorization completed successfully")
	m.mu.Lock()
	session.client = tdClient
	session.state = SessionStateReady
	session.readyAt = time.Now()
	m.mu.Unlock()

	// Authorize() returns without calling Handle for Ready state,
	// so we must emit Ready here for WaitForState to pick it up.
	select {
	case session.authorizer.stateCh <- SessionStateReady:
	default:
	}
}

// WaitForState blocks until the session transitions to a new state or the context is cancelled.
// This is the key method that bridges the async TDLib flow with synchronous REST calls.
// It waits for the authorizer to emit a state change (meaning TDLib is now waiting for input).
func (m *Manager) WaitForState(ctx context.Context, sessionID string, timeout time.Duration) (SessionState, error) {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return "", err
	}

	// If session is already in a terminal or actionable state, return immediately
	m.mu.RLock()
	currentState := session.state
	m.mu.RUnlock()

	if currentState == SessionStateReady || currentState == SessionStateError || currentState == SessionStateClosed {
		return currentState, nil
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case state, ok := <-session.authorizer.stateCh:
		if !ok {
			return SessionStateClosed, nil
		}
		m.mu.Lock()
		session.state = state
		m.mu.Unlock()
		return state, nil
	case <-timer.C:
		return "", fmt.Errorf("timeout waiting for state change")
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// SubmitPhone submits a phone number for a session that is in SessionStateWaitPhone.
func (m *Manager) SubmitPhone(sessionID string, phone string) error {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return err
	}

	m.mu.RLock()
	state := session.state
	m.mu.RUnlock()

	if state != SessionStateWaitPhone {
		return fmt.Errorf("session is not waiting for phone (current state: %s)", state)
	}

	select {
	case session.authorizer.phoneCh <- phone:
		return nil
	default:
		return fmt.Errorf("phone channel is full; a phone number was already submitted")
	}
}

// SubmitCode submits a verification code for a session that is in SessionStateWaitCode.
func (m *Manager) SubmitCode(sessionID string, code string) error {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return err
	}

	m.mu.RLock()
	state := session.state
	m.mu.RUnlock()

	if state != SessionStateWaitCode {
		return fmt.Errorf("session is not waiting for code (current state: %s)", state)
	}

	select {
	case session.authorizer.codeCh <- code:
		return nil
	default:
		return fmt.Errorf("code channel is full; a code was already submitted")
	}
}

// SubmitPassword submits a 2FA password for a session that is in SessionStateWaitPassword.
func (m *Manager) SubmitPassword(sessionID string, password string) error {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return err
	}

	m.mu.RLock()
	state := session.state
	m.mu.RUnlock()

	if state != SessionStateWaitPassword {
		return fmt.Errorf("session is not waiting for password (current state: %s)", state)
	}

	select {
	case session.authorizer.passwordCh <- password:
		return nil
	default:
		return fmt.Errorf("password channel is full; a password was already submitted")
	}
}

// GetSession returns the session by ID or an error if not found.
func (m *Manager) GetSession(sessionID string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return session, nil
}

// GetState returns the current state of a session without blocking.
func (m *Manager) GetState(sessionID string) (SessionState, error) {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return "", err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	return session.state, nil
}

// GetListener returns a TDLib update listener for a ready session.
func (m *Manager) GetListener(sessionID string) (*client.Listener, error) {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if session.state != SessionStateReady || session.client == nil {
		return nil, fmt.Errorf("session is not ready (state: %s)", session.state)
	}

	return session.client.GetListener(), nil
}

// GetClient returns the TDLib client for a ready session.
// Used to call TDLib methods like GetMe() after authentication.
func (m *Manager) GetClient(sessionID string) (*client.Client, error) {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if session.state != SessionStateReady || session.client == nil {
		return nil, fmt.Errorf("session is not ready (state: %s)", session.state)
	}

	return session.client, nil
}

// GetDatabaseDir returns the database directory path for a session.
func (m *Manager) GetDatabaseDir(sessionID string) string {
	return filepath.Join(m.cfg.DatabaseBaseDir, sessionID)
}

// GetAuthTimeout returns the configured auth timeout duration.
func (m *Manager) GetAuthTimeout() time.Duration {
	return m.cfg.AuthTimeout
}

// CloseSession closes a session and releases its resources.
func (m *Manager) CloseSession(sessionID string) error {
	m.mu.Lock()
	session, ok := m.sessions[sessionID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("session not found: %s", sessionID)
	}
	delete(m.sessions, sessionID)
	m.mu.Unlock()

	if session.client != nil {
		_, _ = session.client.Close()
	}

	session.state = SessionStateClosed
	return nil
}

// ListSessions returns info about all active sessions.
func (m *Manager) ListSessions() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		result = append(result, s)
	}
	return result
}
