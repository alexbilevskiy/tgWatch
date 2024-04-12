package libs

import (
	"github.com/alexbilevskiy/tgWatch/pkg/libs/mongo"
	"github.com/alexbilevskiy/tgWatch/pkg/libs/tdlib"
	"github.com/alexbilevskiy/tgWatch/pkg/libs/tdlib/tdAccount"
	"github.com/zelenin/go-tdlib/client"
	"sync"
)

type Account struct {
	Id       int64
	Phone    string
	DbPrefix string
	DataDir  string
	Status   string
	Username string
	TdApi    *tdlib.TdApi
	Me       *client.User
}

func (acc *Account) RunAccount() {
	tdMongo := mongo.TdMongo{}
	tdMongo.Init(acc.DbPrefix, acc.Phone)

	tdlibClient, me := tdAccount.RunTdlib(*acc)
	acc.Username = tdlib.GetUsername(me.Usernames)
	acc.Me = me

	tdApi := tdlib.TdApi{}
	tdApi.Init(acc, tdlibClient, &tdMongo)

	acc.TdApi = &tdApi

	go tdApi.ListenUpdates()
}

var AS AccountStorage

type AccountStorage struct {
	accounts sync.Map
}

func (as *AccountStorage) Init() {
}

func (as *AccountStorage) Store(acc *Account) {
	as.accounts.Store(acc.Id, acc)
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
