package helpers

import (
	"context"
	"go-tdlib/client"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"tgWatch/config"
	"tgWatch/structs"
	"time"
)

var mongoClient *mongo.Client
var mongoContext context.Context
var updatesColl *mongo.Collection
var tdlibClient *client.Client

func Init() {
	config.InitConfiguration()
	initMongo()
	initWeb()
	initTdlib()
}

func initMongo() {
	mongoContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	mongoClient, err = mongo.Connect(mongoContext, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatalf("Mongo error: %s", err)
	}
	updatesColl = mongoClient.Database("tg").Collection("updates")
}

func SaveUpdate(t string, upd interface{}, timestamp int32) string {
	if timestamp == 0 {
		timestamp = int32(time.Now().Unix())
	}
	update := structs.TgUpdate{T: t, Time: timestamp, Upd: upd}
	res, err := updatesColl.InsertOne(mongoContext, update)
	if err != nil {
		log.Printf("[ERROR] insert %s: %s", t, err)

		return ""
	}

	return res.InsertedID.(primitive.ObjectID).String()
}