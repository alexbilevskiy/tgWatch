package account

import (
	"log"
	"sync"
	"time"

	"github.com/alexbilevskiy/tgWatch/internal/consts"
	"github.com/alexbilevskiy/tgWatch/internal/db"
	"go.mongodb.org/mongo-driver/mongo"
)

var AS AccountStore

type AccountStore struct {
	storage     *db.AccountsStorage
	mongoClient *mongo.Client
	accounts    sync.Map
}

func NewAccountsStore(mongoClient *mongo.Client, as *db.AccountsStorage) *AccountStore {
	return &AccountStore{storage: as, mongoClient: mongoClient, accounts: sync.Map{}}
}

func (as *AccountStore) Put(id int64, acc *Account) {
	as.accounts.Store(id, acc)
}

func (as *AccountStore) Get(accId int64) *Account {
	acc, ok := as.accounts.Load(accId)
	if !ok {
		return nil
	}

	return acc.(*Account)
}

func (as *AccountStore) Delete(accId int64) {

	as.accounts.Delete(accId)
}

func (as *AccountStore) Range(f func(key any, value any) bool) {

	as.accounts.Range(f)
}

func (as *AccountStore) RunOne(phone string) {
	accounts := as.storage.LoadAccounts(phone)
	for _, mongoAcc := range accounts {
		if mongoAcc.Status != consts.AccStatusActive {
			log.Printf("wont run Account %d, because its not active yet: `%s`", mongoAcc.Id, mongoAcc.Status)
			continue
		}
		log.Printf("create Account %d", mongoAcc.Id)
		tdm := db.NewTdMongo(as.mongoClient, mongoAcc.DbPrefix, mongoAcc.Phone)
		acc := NewAccount(tdm, mongoAcc)
		AS.Put(mongoAcc.Id, acc)
		AS.Get(mongoAcc.Id).Run()
	}
}

func (as *AccountStore) Run() {
	for {
		accounts := as.storage.LoadAccounts("")
		for _, mongoAcc := range accounts {
			if AS.Get(mongoAcc.Id) != nil {
				if mongoAcc.Status != consts.AccStatusActive {
					//not implemented actually. No one updates status to non-active axcept when upating manually in DB
					log.Printf("need to stop Account %d, because it became active: `%s`", mongoAcc.Id, mongoAcc.Status)
					AS.Get(mongoAcc.Id).TdApi.Close()
					AS.Delete(mongoAcc.Id)
				} else {
					//already running
				}
				continue
			}
			if mongoAcc.Status != consts.AccStatusActive {
				log.Printf("wont run Account %d, because its not active yet: `%s`", mongoAcc.Id, mongoAcc.Status)
				continue
			}
			log.Printf("create Account %d", mongoAcc.Id)
			tdm := db.NewTdMongo(as.mongoClient, mongoAcc.DbPrefix, mongoAcc.Phone)
			acc := NewAccount(tdm, mongoAcc)
			AS.Put(mongoAcc.Id, acc)
			AS.Get(mongoAcc.Id).Run()
		}
		time.Sleep(5 * time.Second)
	}
}
