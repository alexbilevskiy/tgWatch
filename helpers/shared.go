package helpers

import (
	"context"
	"go-tdlib/client"
	"go.mongodb.org/mongo-driver/mongo"
	"tgWatch/config"
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
