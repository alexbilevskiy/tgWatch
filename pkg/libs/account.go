package libs

import (
	"github.com/alexbilevskiy/tgWatch/pkg/consts"
	"github.com/alexbilevskiy/tgWatch/pkg/libs/mongo"
	"github.com/alexbilevskiy/tgWatch/pkg/libs/tdlib"
	"github.com/alexbilevskiy/tgWatch/pkg/libs/tdlib/tdAccount"
	"github.com/zelenin/go-tdlib/client"
	"log"
	"sync"
	"time"
)

type Account struct {
	DbData   *mongo.DbAccountData
	Username string
	TdApi    *tdlib.TdApi
	Me       *client.User
}

func (acc *Account) RunAccount() {
	tdMongo := mongo.TdMongo{}
	tdMongo.Init(acc.DbData.DbPrefix, acc.DbData.Phone)

	tdlibClient, me := tdAccount.RunTdlib(acc.DbData)
	acc.Username = tdlib.GetUsername(me.Usernames)
	acc.Me = me

	tdApi := tdlib.TdApi{}
	tdApi.Init(acc.DbData, tdlibClient, &tdMongo)

	acc.TdApi = &tdApi

	go tdApi.ListenUpdates()
}

var AS AccountStorage

type AccountStorage struct {
	accounts sync.Map
}

func (as *AccountStorage) Create(mongoAcc *mongo.DbAccountData) {
	acc := &Account{
		DbData: mongoAcc,
	}
	as.accounts.Store(acc.DbData.Id, acc)
}

func (as *AccountStorage) Get(accId int64) *Account {
	acc, ok := as.accounts.Load(accId)
	if !ok {
		return nil
	}

	return acc.(*Account)
}

func (as *AccountStorage) Range(f func(key any, value any) bool) {

	as.accounts.Range(f)
}

func (as *AccountStorage) RunOne(phone string) {
	accounts := mongo.LoadAccounts(phone)
	for _, mongoAcc := range accounts {
		if mongoAcc.Status != consts.AccStatusActive {
			log.Printf("wont run account %d, because its not active yet: `%s`", mongoAcc.Id, mongoAcc.Status)
			continue
		}
		log.Printf("create account %d", mongoAcc.Id)
		AS.Create(mongoAcc)
		AS.Get(mongoAcc.Id).RunAccount()
	}
}

func (as *AccountStorage) Run() {
	for {
		accounts := mongo.LoadAccounts("")
		for _, mongoAcc := range accounts {
			if mongoAcc.Status != consts.AccStatusActive {
				log.Printf("wont run account %d, because its not active yet: `%s`", mongoAcc.Id, mongoAcc.Status)
				continue
			}
			if AS.Get(mongoAcc.Id) != nil {
				//already running
				continue
			}
			log.Printf("create account %d", mongoAcc.Id)
			AS.Create(mongoAcc)
			AS.Get(mongoAcc.Id).RunAccount()
		}
		time.Sleep(5 * time.Second)
	}
}
