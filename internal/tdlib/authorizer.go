package tdlib

import (
	"context"
	"log/slog"
	"time"

	"github.com/zelenin/go-tdlib/client"
)

type ClientAuthorizer struct {
	TdlibParameters *client.SetTdlibParametersRequest
	PhoneNumber     chan string
	Code            chan string
	State           chan client.AuthorizationState
	Password        chan string
	log             *slog.Logger

	phoneSet    bool
	codeSet     bool
	passwordSet bool

	AuthorizerState client.AuthorizationState
}

func NewClientAuthorizer(log *slog.Logger, tdlibParameters *client.SetTdlibParametersRequest) *ClientAuthorizer {
	return &ClientAuthorizer{
		log:             log,
		TdlibParameters: tdlibParameters,
		PhoneNumber:     make(chan string, 1),
		Code:            make(chan string, 1),
		State:           make(chan client.AuthorizationState, 10),
		Password:        make(chan string, 1),
	}
}

func (c *ClientAuthorizer) Handle(tdcl *client.Client, state client.AuthorizationState) error {
	ctx, done := context.WithDeadline(context.Background(), time.Now().Add(60*time.Second)) //ignore new context here
	defer done()
	c.State <- state

	switch state.AuthorizationStateConstructor() {
	case client.ConstructorAuthorizationStateWaitTdlibParameters:
		_, err := tdcl.SetTdlibParameters(ctx, c.TdlibParameters)
		return err

	case client.ConstructorAuthorizationStateWaitPhoneNumber:
		_, err := tdcl.SetAuthenticationPhoneNumber(ctx, &client.SetAuthenticationPhoneNumberRequest{
			PhoneNumber: <-c.PhoneNumber,
			Settings: &client.PhoneNumberAuthenticationSettings{
				AllowFlashCall:       false,
				IsCurrentPhoneNumber: false,
				AllowSmsRetrieverApi: false,
			},
		})
		return err

	case client.ConstructorAuthorizationStateWaitEmailAddress:
		panic("unsupported authorization state TypeAuthorizationStateWaitEmailAddress")
	case client.ConstructorAuthorizationStateWaitEmailCode:
		panic("unsupported authorization state TypeAuthorizationStateWaitEmailCode")
	case client.ConstructorAuthorizationStateWaitOtherDeviceConfirmation:
		panic("unsupported authorization state TypeAuthorizationStateWaitOtherDeviceConfirmation")

	case client.ConstructorAuthorizationStateWaitCode:
		_, err := tdcl.CheckAuthenticationCode(ctx, &client.CheckAuthenticationCodeRequest{
			Code: <-c.Code,
		})
		return err

	case client.ConstructorAuthorizationStateWaitRegistration:
		return client.NotSupportedAuthorizationState(state)

	case client.ConstructorAuthorizationStateWaitPassword:
		_, err := tdcl.CheckAuthenticationPassword(ctx, &client.CheckAuthenticationPasswordRequest{
			Password: <-c.Password,
		})
		return err

	case client.ConstructorAuthorizationStateReady:
		return nil

	case client.ConstructorAuthorizationStateLoggingOut:
		return client.NotSupportedAuthorizationState(state)

	case client.ConstructorAuthorizationStateClosing:
		return nil

	case client.ConstructorAuthorizationStateClosed:
		return nil
	}

	return client.NotSupportedAuthorizationState(state)
}

func (c *ClientAuthorizer) ChanInteractor(phone string, nextParams chan string) {
	var ok bool
	var param string

	defer func() {
		c.AuthorizerState = nil
		c.phoneSet = false
		c.codeSet = false
		c.passwordSet = false
	}()

	for {
		c.AuthorizerState, ok = <-c.State
		if !ok {
			c.log.Info("authorization process closed!")

			return
		}
		c.log.Info("new authorization state", "state", c.AuthorizerState.AuthorizationStateConstructor())

		switch c.AuthorizerState.AuthorizationStateConstructor() {
		case client.ConstructorAuthorizationStateWaitPhoneNumber:
			if c.phoneSet == true {
				continue
			}
			c.log.Info("setting phone...")
			c.PhoneNumber <- phone
			c.phoneSet = true

		case client.ConstructorAuthorizationStateWaitCode:
			if c.codeSet == true {
				continue
			}
			c.log.Info("waiting code...")

			select {
			case param, ok = <-nextParams:
				if !ok {
					c.log.Warn("auth channel closed!")
					continue
				}
			}
			c.log.Info("setting code...")
			c.codeSet = true

			c.Code <- param

		case client.ConstructorAuthorizationStateWaitPassword:
			if c.passwordSet == true {
				continue
			}
			c.log.Info("waiting password...")

			select {
			case param, ok = <-nextParams:
				if !ok {
					c.log.Warn("auth channel closed!")
					continue
				}
			}
			c.log.Info("setting password...")
			c.passwordSet = true

			c.Password <- param

		case client.ConstructorAuthorizationStateReady:
			c.log.Info("authorize complete!")

			return
		}

	}
}

func (c *ClientAuthorizer) Close() {
	close(c.PhoneNumber)
	close(c.Code)
	close(c.State)
	close(c.Password)
}
