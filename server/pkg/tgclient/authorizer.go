package tgclient

import (
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zelenin/go-tdlib/client"
)

// clientAuthorizer is a custom AuthorizationStateHandler that bridges
// the blocking TDLib authorization loop with synchronous REST API calls.
//
// It uses buffered channels with timeouts so that the REST handler can
// "send" a value and the authorizer "receives" it within TDLib's Authorize loop.
// A separate stateCh notifies callers about state transitions.
type clientAuthorizer struct {
	tdlibParams *client.SetTdlibParametersRequest
	sessionType SessionType
	phoneCh     chan string
	botTokenCh  chan string
	codeCh      chan string
	passwordCh  chan string
	stateCh     chan SessionState
	errCh       chan error
	timeout     time.Duration
	log         *logrus.Entry
	// cancelCh is closed when the session is abandoned, so a pending wait for user input
	// returns at once and TDLib is closed instead of idling until the input timeout.
	cancelCh   chan struct{}
	cancelOnce sync.Once

	// lastState avoids logging the same TDLib state on every QR re-check.
	lastState string

	// QR login: the latest tg://login link, refreshed by TDLib as tokens expire.
	qrMu        sync.Mutex
	qrLink      string
	qrStartedAt time.Time
}

// qrPollInterval is how often the QR state is re-read. TDLib has no blocking call
// for authorizationStateWaitOtherDeviceConfirmation, and Authorize re-reads the
// state every time Handle returns, so Handle pauses this long before returning.
const qrPollInterval = time.Second

// QrLink returns the current QR login link, or "" before one is issued.
func (a *clientAuthorizer) QrLink() string {
	a.qrMu.Lock()
	defer a.qrMu.Unlock()
	return a.qrLink
}

func newClientAuthorizer(params *client.SetTdlibParametersRequest, sessionID string, sessionType SessionType, timeout time.Duration) *clientAuthorizer {
	return &clientAuthorizer{
		tdlibParams: params,
		sessionType: sessionType,
		phoneCh:     make(chan string, 1),
		botTokenCh:  make(chan string, 1),
		codeCh:      make(chan string, 1),
		passwordCh:  make(chan string, 1),
		stateCh:     make(chan SessionState, 10),
		errCh:       make(chan error, 1),
		timeout:     timeout,
		log:         logrus.WithField("session_id", sessionID),
		cancelCh:    make(chan struct{}),
	}
}

// cancel aborts any pending wait for user input. Safe to call more than once.
func (a *clientAuthorizer) cancel() {
	a.cancelOnce.Do(func() { close(a.cancelCh) })
}

// Handle implements client.AuthorizationStateHandler.
// This method is called by TDLib's Authorize loop for each state transition.
func (a *clientAuthorizer) Handle(c *client.Client, state client.AuthorizationState) error {
	if stateType := state.AuthorizationStateType(); stateType != a.lastState {
		a.lastState = stateType
		a.log.WithField("tdlib_state", stateType).Info("tdlib authorization state")
	}

	switch state.AuthorizationStateType() {
	case client.TypeAuthorizationStateWaitTdlibParameters:
		_, err := c.SetTdlibParameters(a.tdlibParams)
		return err

	case client.TypeAuthorizationStateWaitPhoneNumber:
		// TDLib asks bots for their token in the same state it asks clients for a phone.
		if a.sessionType == SessionTypeBot {
			a.stateCh <- SessionStateWaitBotToken
			token, err := a.waitForInput(a.botTokenCh)
			if err != nil {
				return err
			}
			_, err = c.CheckAuthenticationBotToken(&client.CheckAuthenticationBotTokenRequest{
				Token: token,
			})
			return err
		}

		// QR login replaces the phone number: TDLib moves to WaitOtherDeviceConfirmation with a link.
		if a.sessionType == SessionTypeQr {
			_, err := c.RequestQrCodeAuthentication(&client.RequestQrCodeAuthenticationRequest{
				OtherUserIds: []int64{},
			})
			return err
		}

		a.stateCh <- SessionStateWaitPhone
		phone, err := a.waitForInput(a.phoneCh)
		if err != nil {
			return err
		}
		_, err = c.SetAuthenticationPhoneNumber(&client.SetAuthenticationPhoneNumberRequest{
			PhoneNumber: phone,
			Settings: &client.PhoneNumberAuthenticationSettings{
				AllowFlashCall:       false,
				IsCurrentPhoneNumber: false,
				AllowSmsRetrieverApi: false,
			},
		})
		return err

	case client.TypeAuthorizationStateWaitCode:
		a.stateCh <- SessionStateWaitCode
		code, err := a.waitForInput(a.codeCh)
		if err != nil {
			return err
		}
		_, err = c.CheckAuthenticationCode(&client.CheckAuthenticationCodeRequest{
			Code: code,
		})
		return err

	case client.TypeAuthorizationStateWaitOtherDeviceConfirmation:
		return a.handleQrConfirmation(state.(*client.AuthorizationStateWaitOtherDeviceConfirmation))

	case client.TypeAuthorizationStateWaitPassword:
		a.stateCh <- SessionStateWaitPassword
		password, err := a.waitForInput(a.passwordCh)
		if err != nil {
			return err
		}
		_, err = c.CheckAuthenticationPassword(&client.CheckAuthenticationPasswordRequest{
			Password: password,
		})
		return err

	case client.TypeAuthorizationStateReady:
		// NOTE: In practice, Authorize() checks for Ready BEFORE calling Handle(),
		// so this case may never execute. The Ready signal is sent from runAuth instead.
		a.stateCh <- SessionStateReady
		return nil

	case client.TypeAuthorizationStateClosing:
		return nil

	case client.TypeAuthorizationStateClosed:
		a.stateCh <- SessionStateClosed
		return nil

	default:
		return fmt.Errorf("unsupported authorization state: %s", state.AuthorizationStateType())
	}
}

// handleQrConfirmation stores the latest QR link and waits briefly for the scan.
// After a scan TDLib moves to WaitPassword (2FA accounts) or Ready.
func (a *clientAuthorizer) handleQrConfirmation(state *client.AuthorizationStateWaitOtherDeviceConfirmation) error {
	a.qrMu.Lock()
	first := a.qrStartedAt.IsZero()
	if first {
		a.qrStartedAt = time.Now()
	}
	a.qrLink = state.Link
	startedAt := a.qrStartedAt
	a.qrMu.Unlock()

	if first {
		a.stateCh <- SessionStateWaitQrScan
	}

	if time.Since(startedAt) > a.timeout {
		return fmt.Errorf("timeout waiting for qr code scan")
	}

	select {
	case <-time.After(qrPollInterval):
		return nil
	case <-a.cancelCh:
		return fmt.Errorf("session cancelled")
	}
}

// Close implements client.AuthorizationStateHandler.
// Note: We do NOT close stateCh here because runAuth writes to it after
// Authorize returns (which calls Close). stateCh is closed separately.
func (a *clientAuthorizer) Close() {
	close(a.phoneCh)
	close(a.botTokenCh)
	close(a.codeCh)
	close(a.passwordCh)
}

// waitForInput blocks until a value is received on ch or the timeout expires.
func (a *clientAuthorizer) waitForInput(ch <-chan string) (string, error) {
	timer := time.NewTimer(a.timeout)
	defer timer.Stop()

	select {
	case val, ok := <-ch:
		if !ok {
			return "", fmt.Errorf("input channel closed")
		}
		return val, nil
	case <-timer.C:
		return "", fmt.Errorf("timeout waiting for user input")
	case <-a.cancelCh:
		return "", fmt.Errorf("session cancelled")
	}
}
