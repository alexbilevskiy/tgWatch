package tdlib

import (
	"context"
	"log/slog"

	"github.com/zelenin/go-tdlib/client"

	"github.com/alexbilevskiy/tgWatch/internal/config"
	"github.com/alexbilevskiy/tgWatch/internal/consts"
	"github.com/alexbilevskiy/tgWatch/internal/db"
)

type AccountCreator struct {
	log                   *slog.Logger
	cfg                   *config.Config
	as                    *db.AccountsStorage
	CurrentAuthorizingAcc *db.DbAccountData
	AuthParams            chan string
	Authorizer            *ClientAuthorizer
}

func NewAccountCreator(log *slog.Logger, cfg *config.Config, astorage *db.AccountsStorage) *AccountCreator {
	return &AccountCreator{log: log, cfg: cfg, as: astorage}
}

func (c *AccountCreator) CurrentState() client.AuthorizationState {
	if c.Authorizer == nil {
		return nil
	}
	return c.Authorizer.AuthorizerState
}

func (c *AccountCreator) RunAccountCreationFlow(phone string) {
	ctx := context.Background() // this is intended
	mongoAcc, err := c.as.GetSavedAccount(ctx, phone)
	if err != nil {
		c.log.Error("unable to check if account exists", "phone", phone, "error", err)
		return
	}
	if mongoAcc == nil {
		c.log.Info("starting new account creation", "phone", phone)
		c.CurrentAuthorizingAcc = &db.DbAccountData{
			Phone:    phone,
			DataDir:  ".tdlib" + phone,
			DbPrefix: "tg",
			Status:   consts.AccStatusNew,
		}
		err := c.as.SaveAccount(ctx, c.CurrentAuthorizingAcc)
		if err != nil {
			c.log.Error("save new account", "phone", phone, "error", err)
			c.CurrentAuthorizingAcc.Status = consts.AccStatusError

			return
		}
	} else {
		c.CurrentAuthorizingAcc = mongoAcc
		if c.CurrentAuthorizingAcc.Status == consts.AccStatusActive {
			c.log.Warn("not creating existing account", "phone", phone)

			return
		}
		c.log.Info("continuing account creation", "phone", phone, "state", c.CurrentAuthorizingAcc.Status)
	}
	c.Authorizer = NewClientAuthorizer(c.log, createTdlibParameters(c.cfg, c.CurrentAuthorizingAcc.DataDir))

	go func() {
		var tdlibClientLocal *client.Client
		var meLocal *client.User

		c.log.Info("push tdlib params", "phone", phone)
		_, _ = client.SetLogVerbosityLevel(&client.SetLogVerbosityLevelRequest{
			NewVerbosityLevel: 2,
		})
		c.AuthParams = make(chan string)

		go c.Authorizer.ChanInteractor(phone, c.AuthParams)

		c.log.Info("create authorizing client instance", "phone", phone)

		var err error
		tdlibClientLocal, err = client.NewClient(c.Authorizer)
		if err != nil {
			c.log.Error("NewClient", "phone", phone, "error", err)
			c.CurrentAuthorizingAcc.Status = consts.AccStatusError
			return
		}
		c.log.Info("get version", "phone", phone)

		optionValue, err := tdlibClientLocal.GetOption(&client.GetOptionRequest{
			Name: "version",
		})
		if err != nil {
			c.log.Error("GetOption", "phone", phone, "error", err)
			c.CurrentAuthorizingAcc.Status = consts.AccStatusError
			return
		}

		c.log.Info("TDLib", "phone", phone, "version", optionValue.(*client.OptionValueString).Value)

		meLocal, err = tdlibClientLocal.GetMe(ctx)
		if err != nil {
			c.log.Error("GetMe", "phone", phone, "error", err)
			c.CurrentAuthorizingAcc.Status = consts.AccStatusError
			return
		}
		id := meLocal.Id

		c.log.Info("NEW Me", "phone", phone, "fname", meLocal.FirstName, "lname", meLocal.LastName, "username", GetUsername(meLocal.Usernames))

		c.log.Info("closing authorizing instance", "phone", phone)
		// TODO: need to restart app after successful account creation to load this acc
		_, err = tdlibClientLocal.Close(ctx)
		if err != nil {
			c.log.Error("close authorizing instance", "phone", phone, "error", err)
			c.CurrentAuthorizingAcc.Status = consts.AccStatusError
			return
		}

		c.CurrentAuthorizingAcc.Id = id
		c.CurrentAuthorizingAcc.Status = consts.AccStatusActive

		err = c.as.SaveAccount(ctx, c.CurrentAuthorizingAcc)
		if err != nil {
			c.log.Error("save new account", "phone", phone, "error", err)
			c.CurrentAuthorizingAcc.Status = consts.AccStatusError
			return
		}

		c.CurrentAuthorizingAcc = nil
	}()
}
