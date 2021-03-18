package libs

import (
	"context"
	"go-tdlib/client"
	"go.mongodb.org/mongo-driver/mongo"
	"tgWatch/config"
	"tgWatch/structs"
)

var mongoClient *mongo.Client
var mongoContext context.Context
var updatesColl *mongo.Collection
var chatFiltersColl *mongo.Collection
var chatListColl *mongo.Collection
var tdlibClient *client.Client
var chatFilters []structs.ChatFilter
var localChats map[int64]*client.Chat

func Init() {
	config.InitConfiguration()
	initMongo()
	initWeb()
	initTdlib()
}
