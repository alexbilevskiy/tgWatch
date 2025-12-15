package db

import (
	"context"
	"fmt"
	"time"

	"github.com/alexbilevskiy/tgWatch/internal/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AccountsStorage struct {
	accountColl *mongo.Collection
}

func NewAccountsStorage(cfg *config.Config, dbClient *mongo.Client) *AccountsStorage {
	return &AccountsStorage{
		accountColl: dbClient.Database(cfg.Mongo["db"]).Collection("accounts"),
	}
}

func (as *AccountsStorage) LoadAccounts(ctx context.Context, phone string) ([]*DbAccountData, error) {
	var crit bson.M
	if phone == "" {
		crit = bson.M{}
	} else {
		crit = bson.M{"phone": phone}
	}
	mctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	accountsCursor, err := as.accountColl.Find(mctx, crit)
	if err != nil {
		return nil, fmt.Errorf("load accounts: %w", err)
	}
	var accountsBson []bson.M
	err = accountsCursor.All(mctx, &accountsBson)
	if err != nil {
		return nil, fmt.Errorf("load accounts cursor: %w", err)
	}
	counter := 0
	accs := make([]*DbAccountData, 0)
	for _, accObj := range accountsBson {
		counter++
		acc := &DbAccountData{
			Id:       accObj["id"].(int64),
			Phone:    accObj["phone"].(string),
			DbPrefix: accObj["dbprefix"].(string),
			DataDir:  accObj["datadir"].(string),
			Status:   accObj["status"].(string),
		}
		accs = append(accs, acc)
	}
	//log.Printf("Loaded %d accounts", counter)

	return accs, nil
}

func (as *AccountsStorage) SaveAccount(ctx context.Context, account *DbAccountData) error {
	crit := bson.D{{"phone", account.Phone}}
	update := bson.D{{"$set", account}}
	t := true
	opts := &options.UpdateOptions{Upsert: &t}

	mctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_, err := as.accountColl.UpdateOne(mctx, crit, update, opts)
	if err != nil {
		return fmt.Errorf("save account: %w", err)
	}
	return nil
}

func (as *AccountsStorage) GetSavedAccount(ctx context.Context, phone string) (*DbAccountData, error) {
	var acc *DbAccountData

	mctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	crit := bson.D{{"phone", phone}}
	accObj := as.accountColl.FindOne(mctx, crit)
	if accObj.Err() == mongo.ErrNoDocuments {
		return nil, nil
	} else if accObj.Err() != nil {
		return nil, fmt.Errorf("get account: %w", accObj.Err())
	}
	err := accObj.Decode(&acc)
	if err != nil {
		return nil, fmt.Errorf("decode db account: %w", err)
	}

	return acc, nil
}
