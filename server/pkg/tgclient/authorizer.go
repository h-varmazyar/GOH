package tgclient

import (
	"fmt"
	"time"

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
	phoneCh     chan string
	codeCh      chan string
	passwordCh  chan string
	stateCh     chan SessionState
	errCh       chan error
	timeout     time.Duration
}

func newClientAuthorizer(params *client.SetTdlibParametersRequest, timeout time.Duration) *clientAuthorizer {
	return &clientAuthorizer{
		tdlibParams: params,
		phoneCh:     make(chan string, 1),
		codeCh:      make(chan string, 1),
		passwordCh:  make(chan string, 1),
		stateCh:     make(chan SessionState, 10),
		errCh:       make(chan error, 1),
		timeout:     timeout,
	}
}

// Handle implements client.AuthorizationStateHandler.
// This method is called by TDLib's Authorize loop for each state transition.
func (a *clientAuthorizer) Handle(c *client.Client, state client.AuthorizationState) error {
	switch state.AuthorizationStateType() {
	case client.TypeAuthorizationStateWaitTdlibParameters:
		_, err := c.SetTdlibParameters(a.tdlibParams)
		return err

	case client.TypeAuthorizationStateWaitPhoneNumber:
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

// Close implements client.AuthorizationStateHandler.
// Note: We do NOT close stateCh here because runAuth writes to it after
// Authorize returns (which calls Close). stateCh is closed separately.
func (a *clientAuthorizer) Close() {
	close(a.phoneCh)
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
	}
}
