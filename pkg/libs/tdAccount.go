package libs

import (
	"github.com/alexbilevskiy/tgWatch/pkg/config"
	"github.com/alexbilevskiy/tgWatch/pkg/structs"
	"github.com/zelenin/go-tdlib/client"
	"log"
	"path/filepath"
)

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
func InitTdlib(acc int64) {
	LoadSettings(acc)
	LoadChatFolders(acc)
	loadOptionsList(acc)
	authorizer := client.ClientAuthorizer()
	go client.CliInteractor(authorizer)

	authorizer.TdlibParameters <- createTdlibParameters(Accounts[acc].DataDir)
	logVerbosity := client.WithLogVerbosity(&client.SetLogVerbosityLevelRequest{
		NewVerbosityLevel: 1,
	})

	var err error
	tdlibClient[acc], err = client.NewClient(authorizer, logVerbosity)
	if err != nil {
		log.Fatalf("NewClient error: %s", err)
	}

	optionValue, err := tdlibClient[acc].GetOption(&client.GetOptionRequest{
		Name: "version",
	})
	if err != nil {
		log.Fatalf("GetOption error: %s", err)
	}

	log.Printf("TDLib version: %s", optionValue.(*client.OptionValueString).Value)

	me[acc], err = tdlibClient[acc].GetMe()
	if err != nil {
		log.Fatalf("GetMe error: %s", err)
	}
	accLocal := Accounts[acc]
	accLocal.Username = GetUsername(me[acc].Usernames)
	Accounts[acc] = accLocal

	log.Printf("Me: %s %s [%s]", me[acc].FirstName, me[acc].LastName, GetUsername(me[acc].Usernames))

	//@NOTE: https://github.com/tdlib/td/issues/1005#issuecomment-613839507
	go func() {
		//for true {
		{
			req := &client.SetOptionRequest{Name: "online", Value: &client.OptionValueBoolean{Value: true}}
			ok, err := tdlibClient[acc].SetOption(req)
			if err != nil {
				log.Printf("failed to set online option: %s", err)
			} else {
				log.Printf("Set online status: %s", JsonMarshalStr(ok))
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
}

const AccStatusActive = "active"
const AccStatusNew = "new"

var authParams chan string
var currentAuthorizingAcc *structs.Account

func CreateAccount(phone string) {
	currentAuthorizingAcc = GetSavedAccount(phone)
	if currentAuthorizingAcc == nil {
		log.Printf("Starting new account creation for phone %s", phone)
		currentAuthorizingAcc = &structs.Account{
			Phone:    phone,
			DataDir:  ".tdlib" + phone,
			DbPrefix: "tg",
			Status:   AccStatusNew,
		}
		SaveAccount(currentAuthorizingAcc)
	} else {
		if currentAuthorizingAcc.Status == AccStatusActive {
			log.Printf("Not creating new account again for phone %s", phone)

			return
		}
		log.Printf("Continuing account creation for phone %s from state %s", phone, currentAuthorizingAcc.Status)
	}

	go func() {
		authorizer := ClientAuthorizer()
		var tdlibClientLocal *client.Client
		var meLocal *client.User

		log.Println("push tdlib params")
		authorizer.TdlibParameters <- createTdlibParameters(currentAuthorizingAcc.DataDir)
		logVerbosity := client.WithLogVerbosity(&client.SetLogVerbosityLevelRequest{
			NewVerbosityLevel: 2,
		})
		authParams = make(chan string)

		go ChanInteractor(authorizer, phone, authParams)

		log.Println("create client")

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
		if err != nil {
			log.Fatalf("GetMe error: %s", err)
		}
		me[meLocal.Id] = meLocal
		tdlibClient[meLocal.Id] = tdlibClientLocal

		log.Printf("NEW Me: %s %s [%s]", meLocal.FirstName, meLocal.LastName, GetUsername(meLocal.Usernames))

		//state = nil
		currentAuthorizingAcc.Id = meLocal.Id
		currentAuthorizingAcc.Status = AccStatusActive
		currentAuthorizingAcc.Username = GetUsername(meLocal.Usernames)

		SaveAccount(currentAuthorizingAcc)
		Accounts[meLocal.Id] = *currentAuthorizingAcc

		currentAuthorizingAcc = nil
	}()
}
