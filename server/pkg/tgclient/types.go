package tgclient

import (
	"time"

	"github.com/zelenin/go-tdlib/client"
)

// SessionState represents the current state of a telegram session's authorization flow.
type SessionState string

const (
	SessionStateWaitPhone    SessionState = "wait_phone"
	SessionStateWaitBotToken SessionState = "wait_bot_token"
	SessionStateWaitCode     SessionState = "wait_code"
	SessionStateWaitPassword SessionState = "wait_password"
	SessionStateReady        SessionState = "ready"
	SessionStateClosed       SessionState = "closed"
	SessionStateError        SessionState = "error"
)

// SessionType represents how the session will authenticate.
type SessionType string

const (
	SessionTypeClient SessionType = "client"
	SessionTypeBot    SessionType = "bot"
)

// Config holds the configuration for the session manager.
type Config struct {
	APIID           int32
	APIHash         string
	DatabaseBaseDir string
	AuthTimeout     time.Duration // max time to complete entire auth flow
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		DatabaseBaseDir: ".tdlib",
		AuthTimeout:     5 * time.Minute,
	}
}

// Session represents one telegram account session.
// It encapsulates the TDLib client and authorization state.
type Session struct {
	id          string
	sessionType SessionType
	state       SessionState
	stateErr    string
	client      *client.Client
	authorizer  *clientAuthorizer
	proxy       *ProxyConfig
	createdAt   time.Time
	readyAt     time.Time
}

// ID returns the session identifier.
func (s *Session) ID() string { return s.id }

// Type returns how the session authenticates (client or bot).
func (s *Session) Type() SessionType { return s.sessionType }

// State returns the current authorization state.
func (s *Session) State() SessionState { return s.state }

// Error returns the error message if state is SessionStateError.
func (s *Session) Error() string { return s.stateErr }

// Client returns the underlying TDLib client. Only valid after state becomes Ready.
func (s *Session) Client() *client.Client { return s.client }

// IsReady returns true if the session has completed authorization.
func (s *Session) IsReady() bool { return s.state == SessionStateReady }
