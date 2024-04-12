package tdAccount

import (
	"github.com/alexbilevskiy/tgWatch/pkg/config"
	"github.com/alexbilevskiy/tgWatch/pkg/libs"
	"github.com/alexbilevskiy/tgWatch/pkg/libs/mongo"
	"github.com/alexbilevskiy/tgWatch/pkg/libs/tdlib"
	"github.com/zelenin/go-tdlib/client"
	"log"
	"path/filepath"
)

func RunTdlib(acc libs.Account) (*client.Client, *client.User) {
	authorizer := client.ClientAuthorizer()
	go client.CliInteractor(authorizer)

	authorizer.TdlibParameters <- createTdlibParameters(acc.DataDir)
	logVerbosity := client.WithLogVerbosity(&client.SetLogVerbosityLevelRequest{
		NewVerbosityLevel: 1,
	})
	//client.WithCatchTimeout(60)

	tdlibClient, err := client.NewClient(authorizer, logVerbosity)
	if err != nil {
		log.Fatalf("NewClient error: %s", err)
	}

	optionValue, err := tdlibClient.GetOption(&client.GetOptionRequest{
		Name: "version",
	})
	if err != nil {
		log.Fatalf("GetOption error: %s", err)
	}

	log.Printf("TDLib version: %s", optionValue.(*client.OptionValueString).Value)

	me, err := tdlibClient.GetMe()
	if err != nil {
		log.Fatalf("GetMe error: %s", err)
	}

	log.Printf("Me: %s %s [%s]", me.FirstName, me.LastName, tdlib.GetUsername(me.Usernames))

	//@NOTE: https://github.com/tdlib/td/issues/1005#issuecomment-613839507
	go func() {
		//for true {
		{
			req := &client.SetOptionRequest{Name: "online", Value: &client.OptionValueBoolean{Value: true}}
			ok, err := tdlibClient.SetOption(req)
			if err != nil {
				log.Printf("failed to set online option: %s", err)
			} else {
				log.Printf("Set online status: %s", libs.JsonMarshalStr(ok))
			}
			//time.Sleep(10 * time.Second)
		}
	}()

	//req := &client.SetOptionRequest{Name: "ignore_background_updates", Value: &client.OptionValueBoolean{Value: false}}
	//ok, err := tdlibClient[acc].SetOption(req)
	//if err != nil {
	//	log.Printf("failed to set ignore_background_updates option: %s", err)
	//} else {
	//	log.Printf("Set ignore_background_updates option: %s", JsonMarshalStr(ok))
	//}

	return tdlibClient, me
}

var AuthParams chan string
var CurrentAuthorizingAcc *libs.Account

func CreateAccount(phone string) {
	CurrentAuthorizingAcc = mongo.GetSavedAccount(phone)
	if CurrentAuthorizingAcc == nil {
		log.Printf("Starting new account creation for phone %s", phone)
		CurrentAuthorizingAcc = &libs.Account{
			Phone:    phone,
			DataDir:  ".tdlib" + phone,
			DbPrefix: "tg",
			Status:   tdlib.AccStatusNew,
		}
		mongo.SaveAccount(CurrentAuthorizingAcc)
	} else {
		if CurrentAuthorizingAcc.Status == tdlib.AccStatusActive {
			log.Printf("Not creating new account again for phone %s", phone)

			return
		}
		log.Printf("Continuing account creation for phone %s from state %s", phone, CurrentAuthorizingAcc.Status)
	}

	go func() {
		authorizer := ClientAuthorizer()
		var tdlibClientLocal *client.Client
		var meLocal *client.User

		log.Println("push tdlib params")
		authorizer.TdlibParameters <- createTdlibParameters(CurrentAuthorizingAcc.DataDir)
		logVerbosity := client.WithLogVerbosity(&client.SetLogVerbosityLevelRequest{
			NewVerbosityLevel: 2,
		})
		AuthParams = make(chan string)

		go ChanInteractor(authorizer, phone, AuthParams)

		log.Println("create authorizing client instance")

		var err error
		tdlibClientLocal, err = client.NewClient(authorizer, logVerbosity)
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

		meLocal, err = tdlibClientLocal.GetMe()
		id := meLocal.Id
		phoneLocal := meLocal.PhoneNumber
		if err != nil {
			log.Fatalf("GetMe error: %s", err)
		}

		log.Printf("NEW Me: %s %s [%s]", meLocal.FirstName, meLocal.LastName, tdlib.GetUsername(meLocal.Usernames))

		CurrentAuthorizingAcc.Id = id
		CurrentAuthorizingAcc.Status = tdlib.AccStatusActive
		CurrentAuthorizingAcc.Username = tdlib.GetUsername(meLocal.Usernames)

		mongo.SaveAccount(CurrentAuthorizingAcc)

		log.Printf("closing authorizing instance")
		_, err = tdlibClientLocal.Close()
		if err != nil {
			log.Printf("failed to close authorizing instance: %s", err.Error())
		}

		CurrentAuthorizingAcc = nil

		log.Printf("create normal client instance for new account %d", id)

		mongo.LoadAccounts(phoneLocal)
		libs.AS.Get(id).RunAccount()
	}()
}

func createTdlibParameters(dataDir string) *client.SetTdlibParametersRequest {
	return &client.SetTdlibParametersRequest{
		UseTestDc:              false,
		DatabaseDirectory:      filepath.Join(config.Config.TDataDir, dataDir, "database"),
		FilesDirectory:         filepath.Join(config.Config.TDataDir, dataDir, "files"),
		UseFileDatabase:        true,
		UseChatInfoDatabase:    true,
		UseMessageDatabase:     true,
		UseSecretChats:         false,
		ApiId:                  config.Config.ApiId,
		ApiHash:                config.Config.ApiHash,
		SystemLanguageCode:     "en",
		DeviceModel:            "Linux",
		SystemVersion:          "1.0.0",
		ApplicationVersion:     "1.0.0",
		EnableStorageOptimizer: true,
		IgnoreFileNames:        false,
	}
}
