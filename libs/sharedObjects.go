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
var settingsColl *mongo.Collection
var tdlibClient *client.Client
var tdlibOptions map[string]structs.TdlibOption
var chatFilters []structs.ChatFilter
var ignoreLists structs.IgnoreLists
var localChats map[int64]*client.Chat
var me *client.User

func Init() {
	config.InitConfiguration()
	initMongo()
	initWeb()
	initTdlib()
}
