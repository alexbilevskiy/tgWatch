package libs

import (
	"context"
	"go-tdlib/client"
	"go.mongodb.org/mongo-driver/mongo"
	"tgWatch/structs"
)

var Accounts map[int64]structs.Account

var mongoClient *mongo.Client
var mongoContext context.Context
var accountColl *mongo.Collection

var updatesColl map[int64]*mongo.Collection
var chatFiltersColl map[int64]*mongo.Collection
var chatListColl map[int64]*mongo.Collection
var settingsColl map[int64]*mongo.Collection
var tdlibClient map[int64]*client.Client
var tdlibOptions map[int64]map[string]structs.TdlibOption
var chatFilters map[int64][]structs.ChatFilter
var ignoreLists map[int64](structs.IgnoreLists)
var localChats map[int64]map[int64]*client.Chat
var me map[int64]*client.User

func InitSharedVars() {
	updatesColl = make(map[int64]*mongo.Collection)
	chatFiltersColl = make(map[int64]*mongo.Collection)
	chatListColl = make(map[int64]*mongo.Collection)
	settingsColl = make(map[int64]*mongo.Collection)
	tdlibClient = make(map[int64]*client.Client)
	tdlibOptions = make(map[int64]map[string]structs.TdlibOption)
	chatFilters = make(map[int64][]structs.ChatFilter)
	ignoreLists = make(map[int64](structs.IgnoreLists))
	localChats = make(map[int64](map[int64]*client.Chat))
	me = make(map[int64]*client.User)
}

func InitSharedSubVars(acc int64) {
	tdlibOptions[acc] = make(map[string]structs.TdlibOption)
	localChats[acc] = make(map[int64]*client.Chat)
}
