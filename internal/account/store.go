package account

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/alexbilevskiy/tgWatch/internal/config"
	"github.com/alexbilevskiy/tgWatch/internal/consts"
	"github.com/alexbilevskiy/tgWatch/internal/db"
	"go.mongodb.org/mongo-driver/mongo"
)

type AccountsStore struct {
	cfg         *config.Config
	storage     *db.AccountsStorage
	mongoClient *mongo.Client
	accounts    *sync.Map
}

func NewAccountsStore(cfg *config.Config, mongoClient *mongo.Client, as *db.AccountsStorage) *AccountsStore {
	return &AccountsStore{cfg: cfg, storage: as, mongoClient: mongoClient, accounts: &sync.Map{}}
}

func (as *AccountsStore) Put(id int64, acc *Account) {
	as.accounts.Store(id, acc)
}

func (as *AccountsStore) Get(accId int64) *Account {
	acc, ok := as.accounts.Load(accId)
	if !ok {
		return nil
	}

	return acc.(*Account)
}

func (as *AccountsStore) Delete(accId int64) {

	as.accounts.Delete(accId)
}

func (as *AccountsStore) Range(f func(key any, value any) bool) {

	as.accounts.Range(f)
}

func (as *AccountsStore) RunOne(ctx context.Context, phone string) error {
	accounts, err := as.storage.LoadAccounts(ctx, phone)
	if err != nil {
		return fmt.Errorf("run one account: %w", err.Error())
	}
	for _, mongoAcc := range accounts {
		if mongoAcc.Status != consts.AccStatusActive {
			log.Printf("wont run Account %d, because its not active yet: `%s`", mongoAcc.Id, mongoAcc.Status)
			continue
		}
		log.Printf("create Account %d", mongoAcc.Id)
		tdm := db.NewTdMongo(as.mongoClient, mongoAcc.DbPrefix, mongoAcc.Phone)
		acc := NewAccount(as.cfg, tdm, mongoAcc)
		err := acc.Run(ctx)
		if err != nil {
			log.Printf("failed to run account: %s", err.Error())
			continue
		}
		as.Put(mongoAcc.Id, acc)
	}

	return nil
}

func (as *AccountsStore) Run(ctx context.Context) error {
	accounts, err := as.storage.LoadAccounts(ctx, "")
	if err != nil {
		return fmt.Errorf("run accounts: %w", err)
	}
	for _, mongoAcc := range accounts {
		if as.Get(mongoAcc.Id) != nil {
			if mongoAcc.Status != consts.AccStatusActive {
				//not implemented actually. No one updates status to non-active axcept when upating manually in DB
				log.Printf("need to stop Account %d, because it became active: `%s`", mongoAcc.Id, mongoAcc.Status)
				as.Get(mongoAcc.Id).TdApi.Close(ctx)
				as.Delete(mongoAcc.Id)
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
		acc := NewAccount(as.cfg, tdm, mongoAcc)
		err := acc.Run(ctx)
		if err != nil {
			log.Printf("failed to run account: %s", err.Error())
			continue
		}
		as.Put(mongoAcc.Id, acc)
	}
	return nil
}
