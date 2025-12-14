package db

import (
	"context"
	"fmt"
	"log"
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
	as := &AccountsStorage{}
	as.accountColl = dbClient.Database(cfg.Mongo["db"]).Collection("accounts")
	return as
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

func (as *AccountsStorage) SaveAccount(ctx context.Context, account *DbAccountData) {
	crit := bson.D{{"phone", account.Phone}}
	update := bson.D{{"$set", account}}
	t := true
	opts := &options.UpdateOptions{Upsert: &t}

	mctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_, err := as.accountColl.UpdateOne(mctx, crit, update, opts)
	if err != nil {
		log.Fatalf("Failed to save account %d", account.Id)
	}
	log.Printf("Saved new account id:%d", account.Id)
}

func (as *AccountsStorage) GetSavedAccount(ctx context.Context, phone string) *DbAccountData {
	var acc *DbAccountData

	mctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	crit := bson.D{{"phone", phone}}
	accObj := as.accountColl.FindOne(mctx, crit)
	if accObj.Err() == mongo.ErrNoDocuments {
		return nil
	} else if accObj.Err() != nil {
		log.Fatalf("Failed to find account: %s", accObj.Err().Error())
	}
	err := accObj.Decode(&acc)
	if err != nil {
		log.Fatalf("Failed to decode db account: %s", accObj.Err().Error())
	}

	return acc
}
