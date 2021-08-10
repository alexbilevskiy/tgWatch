package libs

import (
	"context"
	"go-tdlib/client"
	"go.mongodb.org/mongo-driver/mongo"
	"tgWatch/structs"
)

var Accounts map[int32]structs.Account

var mongoClient *mongo.Client
var mongoContext context.Context
var accountColl *mongo.Collection

var updatesColl map[int32]*mongo.Collection
var chatFiltersColl map[int32]*mongo.Collection
var chatListColl map[int32]*mongo.Collection
var settingsColl map[int32]*mongo.Collection
var tdlibClient map[int32]*client.Client
var tdlibOptions map[int32]map[string]structs.TdlibOption
var chatFilters map[int32][]structs.ChatFilter
var ignoreLists map[int32](structs.IgnoreLists)
var localChats map[int32]map[int64]*client.Chat
var me map[int32]*client.User

func InitSharedVars() {
	updatesColl = make(map[int32]*mongo.Collection)
	chatFiltersColl = make(map[int32]*mongo.Collection)
	chatListColl = make(map[int32]*mongo.Collection)
	settingsColl = make(map[int32]*mongo.Collection)
	tdlibClient = make(map[int32]*client.Client)
	tdlibOptions = make(map[int32]map[string]structs.TdlibOption)
	chatFilters = make(map[int32][]structs.ChatFilter)
	ignoreLists = make(map[int32](structs.IgnoreLists))
	localChats = make(map[int32](map[int64]*client.Chat))
	me = make(map[int32]*client.User)
}

func InitSharedSubVars(acc int32) {
	tdlibOptions[acc] = make(map[string]structs.TdlibOption)
	localChats[acc] = make(map[int64]*client.Chat)
}