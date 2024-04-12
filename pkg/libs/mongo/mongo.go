package mongo

import (
	"context"
	"github.com/alexbilevskiy/tgWatch/pkg/config"
	"github.com/alexbilevskiy/tgWatch/pkg/libs"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"
)

var mongoClient *mongo.Client
var mongoContext context.Context

var accountColl *mongo.Collection

func InitGlobalMongo() {
	var cancel context.CancelFunc
	mongoContext, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rb := bson.NewRegistryBuilder()

	//@TODO: see messageSenderDecoder above
	//var a *client.MessageSender
	//rb.RegisterHookDecoder(reflect.TypeOf(a).Elem(), messageSenderDecoder{})
	//rb.RegisterTypeDecoder(reflect.TypeOf((client.MessageSender)(nil)).Elem(), messageSenderDecoder{})
	//rb.RegisterTypeDecoder(reflect.TypeOf(client.MessageSenderChat{})	, messageSenderDecoder{})

	registry := rb.Build()
	clientOptions := options.Client().ApplyURI(config.Config.Mongo["uri"]).SetRegistry(registry)

	var err error
	mongoClient, err = mongo.Connect(mongoContext, clientOptions)
	if err != nil {
		log.Fatalf("Mongo error: %s", err)
		return
	}
	//@TODO: why we need context on each query and why it is possible to use null?
	mongoContext = nil
	accountColl = mongoClient.Database(config.Config.Mongo["db"]).Collection("accounts")
}

func LoadAccounts(phone string) {
	var crit bson.M
	if phone == "" {
		crit = bson.M{}
	} else {
		crit = bson.M{"phone": phone}
	}
	accountsCursor, err := accountColl.Find(mongoContext, crit)
	if err != nil {
		log.Fatalf("Accounts load error: %s", err.Error())
		return
	}
	var accountsBson []bson.M
	err = accountsCursor.All(mongoContext, &accountsBson)
	if err != nil {
		log.Fatalf("Accounts cursor error: %s", err.Error())
		return
	}
	counter := 0
	for _, accObj := range accountsBson {
		counter++
		acc := libs.Account{
			Id:       accObj["id"].(int64),
			Phone:    accObj["phone"].(string),
			DbPrefix: accObj["dbprefix"].(string),
			DataDir:  accObj["datadir"].(string),
			Status:   accObj["status"].(string),
		}
		libs.AS.Store(&acc)
	}
	log.Printf("Loaded %d accounts", counter)
}

func SaveAccount(account *libs.Account) {
	crit := bson.D{{"phone", account.Phone}}
	update := bson.D{{"$set", account}}
	t := true
	opts := &options.UpdateOptions{Upsert: &t}

	_, err := accountColl.UpdateOne(mongoContext, crit, update, opts)
	if err != nil {
		log.Fatalf("Failed to save account %d", account.Id)
	}
	log.Printf("Saved new account id:%d", account.Id)
}

func GetSavedAccount(phone string) *libs.Account {
	var acc *libs.Account

	crit := bson.D{{"phone", phone}}
	accObj := accountColl.FindOne(mongoContext, crit)
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
