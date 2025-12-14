package tdlib

import (
	"context"
	"log"

	"github.com/alexbilevskiy/tgWatch/internal/config"
	"github.com/alexbilevskiy/tgWatch/internal/consts"
	"github.com/alexbilevskiy/tgWatch/internal/db"
	"github.com/zelenin/go-tdlib/client"
)

type AccountCreator struct {
	cfg                   *config.Config
	as                    *db.AccountsStorage
	CurrentAuthorizingAcc *db.DbAccountData
	AuthParams            chan string
}

func NewAccountCreator(cfg *config.Config, astorage *db.AccountsStorage) *AccountCreator {
	return &AccountCreator{cfg: cfg, as: astorage}
}

func (c *AccountCreator) CreateAccount(ctx context.Context, phone string) {
	mongoAcc := c.as.GetSavedAccount(ctx, phone)
	if mongoAcc == nil {
		log.Printf("Starting new account creation for phone %s", phone)
		c.CurrentAuthorizingAcc = &db.DbAccountData{
			Phone:    phone,
			DataDir:  ".tdlib" + phone,
			DbPrefix: "tg",
			Status:   consts.AccStatusNew,
		}
		c.as.SaveAccount(ctx, c.CurrentAuthorizingAcc)
	} else {
		c.CurrentAuthorizingAcc = mongoAcc
		if c.CurrentAuthorizingAcc.Status == consts.AccStatusActive {
			log.Printf("Not creating new account again for phone %s", phone)

			return
		}
		log.Printf("Continuing account creation for phone %s from state %s", phone, c.CurrentAuthorizingAcc.Status)
	}

	go func() {
		authorizer := ClientAuthorizer(createTdlibParameters(c.cfg, c.CurrentAuthorizingAcc.DataDir))
		var tdlibClientLocal *client.Client
		var meLocal *client.User

		log.Println("push tdlib params")
		_, _ = client.SetLogVerbosityLevel(&client.SetLogVerbosityLevelRequest{
			NewVerbosityLevel: 2,
		})
		c.AuthParams = make(chan string)

		go ChanInteractor(authorizer, phone, c.AuthParams)

		log.Println("create authorizing client instance")

		var err error
		tdlibClientLocal, err = client.NewClient(authorizer)
		if err != nil {
			log.Fatalf("NewClient error: %s", err)
		}
		log.Println("get version")

		optionValue, err := tdlibClientLocal.GetOption(&client.GetOptionRequest{
			Name: "version",
		})
		if err != nil {
			log.Fatalf("GetOption error: %s", err)
		}

		log.Printf("TDLib version: %s", optionValue.(*client.OptionValueString).Value)

		meLocal, err = tdlibClientLocal.GetMe(ctx)
		id := meLocal.Id
		if err != nil {
			log.Fatalf("GetMe error: %s", err)
		}

		log.Printf("NEW Me: %s %s [%s]", meLocal.FirstName, meLocal.LastName, GetUsername(meLocal.Usernames))

		log.Printf("closing authorizing instance")
		_, err = tdlibClientLocal.Close(ctx)

		c.CurrentAuthorizingAcc.Id = id
		c.CurrentAuthorizingAcc.Status = consts.AccStatusActive

		c.as.SaveAccount(ctx, c.CurrentAuthorizingAcc)

		if err != nil {
			log.Printf("failed to close authorizing instance: %s", err.Error())
		}

		c.CurrentAuthorizingAcc = nil
	}()
}
