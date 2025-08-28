package tdlib

import (
	"context"
	"log"
	"time"

	"github.com/zelenin/go-tdlib/client"
)

type clientAuthorizer struct {
	TdlibParameters *client.SetTdlibParametersRequest
	PhoneNumber     chan string
	Code            chan string
	State           chan client.AuthorizationState
	Password        chan string
}

func (stateHandler *clientAuthorizer) Handle(tdcl *client.Client, state client.AuthorizationState) error {
	ctx, done := context.WithDeadline(context.Background(), time.Now().Add(60*time.Second))
	defer done()
	stateHandler.State <- state

	switch state.AuthorizationStateConstructor() {
	case client.ConstructorAuthorizationStateWaitTdlibParameters:
		_, err := tdcl.SetTdlibParameters(ctx, stateHandler.TdlibParameters)
		return err

	case client.ConstructorAuthorizationStateWaitPhoneNumber:
		_, err := tdcl.SetAuthenticationPhoneNumber(ctx, &client.SetAuthenticationPhoneNumberRequest{
			PhoneNumber: <-stateHandler.PhoneNumber,
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
			Code: <-stateHandler.Code,
		})
		return err

	case client.ConstructorAuthorizationStateWaitRegistration:
		return client.NotSupportedAuthorizationState(state)

	case client.ConstructorAuthorizationStateWaitPassword:
		_, err := tdcl.CheckAuthenticationPassword(ctx, &client.CheckAuthenticationPasswordRequest{
			Password: <-stateHandler.Password,
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

func (stateHandler *clientAuthorizer) Close() {
	close(stateHandler.PhoneNumber)
	close(stateHandler.Code)
	close(stateHandler.State)
	close(stateHandler.Password)
}

func ClientAuthorizer(tdlibParameters *client.SetTdlibParametersRequest) *clientAuthorizer {
	return &clientAuthorizer{
		TdlibParameters: tdlibParameters,
		PhoneNumber:     make(chan string, 1),
		Code:            make(chan string, 1),
		State:           make(chan client.AuthorizationState, 10),
		Password:        make(chan string, 1),
	}
}

var AuthorizerState client.AuthorizationState
var phoneSet bool = false
var codeSet bool = false
var passwordSet bool = false

func ChanInteractor(clientAuthorizer *clientAuthorizer, phone string, nextParams chan string) {
	var ok bool
	var param string

	defer func() {
		AuthorizerState = nil
		phoneSet = false
		codeSet = false
		passwordSet = false
	}()

	for {
		AuthorizerState, ok = <-clientAuthorizer.State
		if !ok {
			log.Printf("Authorization process closed!")

			return
		}
		log.Printf("new state! %s", AuthorizerState.AuthorizationStateConstructor())

		switch AuthorizerState.AuthorizationStateConstructor() {
		case client.ConstructorAuthorizationStateWaitPhoneNumber:
			if phoneSet == true {
				continue
			}
			log.Printf("Setting phone...")
			clientAuthorizer.PhoneNumber <- phone
			phoneSet = true

		case client.ConstructorAuthorizationStateWaitCode:
			if codeSet == true {
				continue
			}
			log.Printf("Waiting code...")

			select {
			case param, ok = <-nextParams:
				if !ok {
					log.Printf("Invalid param!")
					continue
				}
			}
			log.Printf("Setting code...")
			codeSet = true

			clientAuthorizer.Code <- param

		case client.ConstructorAuthorizationStateWaitPassword:
			if passwordSet == true {
				continue
			}
			log.Printf("Waiting password...")

			select {
			case param, ok = <-nextParams:
				if !ok {
					log.Printf("Invalid param!")
					continue
				}
			}
			log.Printf("Setting password...")
			passwordSet = true

			clientAuthorizer.Password <- param

		case client.ConstructorAuthorizationStateReady:
			log.Printf("Authorize complete!")

			return
		}

	}
}
