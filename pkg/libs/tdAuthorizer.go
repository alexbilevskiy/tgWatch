package libs

import (
	"fmt"
	"github.com/zelenin/go-tdlib/client"
	"time"
)

type clientAuthorizer struct {
	TdlibParameters chan *client.SetTdlibParametersRequest
	PhoneNumber     chan string
	Code            chan string
	State           chan client.AuthorizationState
	Password        chan string
}

func (stateHandler *clientAuthorizer) Handle(tdcl *client.Client, state client.AuthorizationState) error {
	stateHandler.State <- state

	switch state.AuthorizationStateType() {
	case client.TypeAuthorizationStateWaitTdlibParameters:
		_, err := tdcl.SetTdlibParameters(<-stateHandler.TdlibParameters)
		return err

	case client.TypeAuthorizationStateWaitPhoneNumber:
		_, err := tdcl.SetAuthenticationPhoneNumber(&client.SetAuthenticationPhoneNumberRequest{
			PhoneNumber: <-stateHandler.PhoneNumber,
			Settings: &client.PhoneNumberAuthenticationSettings{
				AllowFlashCall:       false,
				IsCurrentPhoneNumber: false,
				AllowSmsRetrieverApi: false,
			},
		})
		return err

	case client.TypeAuthorizationStateWaitEmailAddress:
		panic("unsupported authorization state TypeAuthorizationStateWaitEmailAddress")
	case client.TypeAuthorizationStateWaitEmailCode:
		panic("unsupported authorization state TypeAuthorizationStateWaitEmailCode")
	case client.TypeAuthorizationStateWaitOtherDeviceConfirmation:
		panic("unsupported authorization state TypeAuthorizationStateWaitOtherDeviceConfirmation")

	case client.TypeAuthorizationStateWaitCode:
		_, err := tdcl.CheckAuthenticationCode(&client.CheckAuthenticationCodeRequest{
			Code: <-stateHandler.Code,
		})
		return err

	case client.TypeAuthorizationStateWaitRegistration:
		return client.ErrNotSupportedAuthorizationState

	case client.TypeAuthorizationStateWaitPassword:
		_, err := tdcl.CheckAuthenticationPassword(&client.CheckAuthenticationPasswordRequest{
			Password: <-stateHandler.Password,
		})
		return err

	case client.TypeAuthorizationStateReady:
		return nil

	case client.TypeAuthorizationStateLoggingOut:
		return client.ErrNotSupportedAuthorizationState

	case client.TypeAuthorizationStateClosing:
		return nil

	case client.TypeAuthorizationStateClosed:
		return nil
	}

	return client.ErrNotSupportedAuthorizationState
}

func (stateHandler *clientAuthorizer) Close() {
	close(stateHandler.TdlibParameters)
	close(stateHandler.PhoneNumber)
	close(stateHandler.Code)
	close(stateHandler.State)
	close(stateHandler.Password)
}

func ClientAuthorizer() *clientAuthorizer {
	return &clientAuthorizer{
		TdlibParameters: make(chan *client.SetTdlibParametersRequest, 1),
		PhoneNumber:     make(chan string, 1),
		Code:            make(chan string, 1),
		State:           make(chan client.AuthorizationState, 10),
		Password:        make(chan string, 1),
	}
}

var state client.AuthorizationState
var phoneSet bool = false
var codeSet bool = false
var passwordSet bool = false

func ChanInteractor(clientAuthorizer *clientAuthorizer, phone string, nextParams chan string) {
	var ok bool
	var param string
	for {
		if len(clientAuthorizer.State) == 0 {
			if state == nil {
				fmt.Printf("waiting state...\n")
				time.Sleep(1 * time.Second)
				continue
			}
		} else {
			state, ok = <-clientAuthorizer.State
			if !ok {
				fmt.Printf("invalid state...\n")
				time.Sleep(1 * time.Second)

				continue
			}
			fmt.Printf("new state! %s\n", state.AuthorizationStateType())
		}

		switch state.AuthorizationStateType() {
		case client.TypeAuthorizationStateWaitPhoneNumber:
			if phoneSet == true {
				continue
			}
			fmt.Printf("Setting phone...\n")
			clientAuthorizer.PhoneNumber <- phone
			phoneSet = true

		case client.TypeAuthorizationStateWaitCode:
			if codeSet == true {
				continue
			}
			fmt.Printf("Waiting code...\n")

			select {
			case param, ok = <-nextParams:
				if !ok {
					fmt.Printf("Invalid param!\n")
					continue
				}
			}
			fmt.Printf("Setting code...\n")
			codeSet = true

			clientAuthorizer.Code <- param

		case client.TypeAuthorizationStateWaitPassword:
			if passwordSet == true {
				continue
			}
			fmt.Printf("Waiting password...\n")

			select {
			case param, ok = <-nextParams:
				if !ok {
					fmt.Printf("Invalid param!\n")
					continue
				}
			}
			fmt.Printf("Setting password...\n")
			passwordSet = true

			clientAuthorizer.Password <- param

		case client.TypeAuthorizationStateReady:
			state = nil
			phoneSet = false
			codeSet = false
			passwordSet = false

			return
		}

	}
}
