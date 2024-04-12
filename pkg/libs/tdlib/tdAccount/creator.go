package tdAccount

import (
	"github.com/alexbilevskiy/tgWatch/pkg/config"
	"github.com/alexbilevskiy/tgWatch/pkg/consts"
	"github.com/alexbilevskiy/tgWatch/pkg/libs/helpers"
	"github.com/alexbilevskiy/tgWatch/pkg/libs/mongo"
	"github.com/alexbilevskiy/tgWatch/pkg/libs/tdlib"
	"github.com/zelenin/go-tdlib/client"
	"log"
	"path/filepath"
)

func RunTdlib(dbData *mongo.DbAccountData) (*client.Client, *client.User) {
	authorizer := client.ClientAuthorizer()
	go client.CliInteractor(authorizer)

	authorizer.TdlibParameters <- createTdlibParameters(dbData.DataDir)
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
				log.Printf("Set online status: %s", helpers.JsonMarshalStr(ok))
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
var CurrentAuthorizingAcc *mongo.DbAccountData

func CreateAccount(phone string) {
	mongoAcc := mongo.GetSavedAccount(phone)
	if mongoAcc == nil {
		log.Printf("Starting new account creation for phone %s", phone)
		CurrentAuthorizingAcc = &mongo.DbAccountData{
			Phone:    phone,
			DataDir:  ".tdlib" + phone,
			DbPrefix: "tg",
			Status:   consts.AccStatusNew,
		}
		mongo.SaveAccount(CurrentAuthorizingAcc)
	} else {
		CurrentAuthorizingAcc = mongoAcc
		if CurrentAuthorizingAcc.Status == consts.AccStatusActive {
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
		if err != nil {
			log.Fatalf("GetMe error: %s", err)
		}

		log.Printf("NEW Me: %s %s [%s]", meLocal.FirstName, meLocal.LastName, tdlib.GetUsername(meLocal.Usernames))

		CurrentAuthorizingAcc.Id = id
		CurrentAuthorizingAcc.Status = consts.AccStatusActive

		mongo.SaveAccount(CurrentAuthorizingAcc)

		log.Printf("closing authorizing instance")
		_, err = tdlibClientLocal.Close()
		if err != nil {
			log.Printf("failed to close authorizing instance: %s", err.Error())
		}

		CurrentAuthorizingAcc = nil

		//@TODO: does not work again!!!
		//log.Printf("create normal client instance for new account %d", id)
		//mongo.LoadAccounts(meLocal.PhoneNumber)
		//libs.AS.Get(id).RunAccount()
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
