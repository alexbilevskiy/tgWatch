package mongo

import (
	"fmt"
	"github.com/alexbilevskiy/tgWatch/pkg/consts"
	"github.com/alexbilevskiy/tgWatch/pkg/structs"
	"github.com/zelenin/go-tdlib/client"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
)

type TdMongo struct {
	updatesColl     *mongo.Collection
	chatFiltersColl *mongo.Collection
	chatListColl    *mongo.Collection
	settingsColl    *mongo.Collection
	settings        structs.IgnoreLists
}

type TdStorageInterface interface {
	Init(DbPrefix string, Phone string)
	DeleteChatFolder(folderId int32) (*mongo.DeleteResult, error)
	ClearChatFilters()
	LoadChatFolders() []structs.ChatFilter
	GetSettings() structs.IgnoreLists

	SaveChatFolder(chatFolder *client.ChatFolder, folderInfo *client.ChatFolderInfo)
	SaveAllChatPositions(chatId int64, positions []*client.ChatPosition)
	SaveChatPosition(chatId int64, chatPosition *client.ChatPosition)

	GetSavedChats(listId int32) []structs.ChatPosition

	loadSettings()
	SaveSettings(il structs.IgnoreLists)
}

func (m *TdMongo) Init(DbPrefix string, Phone string) {
	db := DbPrefix + Phone
	m.updatesColl = mongoClient.Database(db).Collection("updates")
	m.chatFiltersColl = mongoClient.Database(db).Collection("chatFilters")
	m.chatListColl = mongoClient.Database(db).Collection("chatList")
	m.settingsColl = mongoClient.Database(db).Collection("settings")

	m.loadSettings()
	m.LoadChatFolders()
}

func (m *TdMongo) SaveChatFolder(chatFolder *client.ChatFolder, folderInfo *client.ChatFolderInfo) {

	filStr := structs.ChatFilter{Id: folderInfo.Id, Title: folderInfo.Title, IncludedChats: chatFolder.IncludedChatIds}
	crit := bson.D{{"id", folderInfo.Id}}
	update := bson.D{{"$set", filStr}}
	t := true
	opts := &options.UpdateOptions{Upsert: &t}
	_, err := m.chatFiltersColl.UpdateOne(mongoContext, crit, update, opts)
	if err != nil {
		fmt.Printf("Failed to save chat filter: id: %d, n: %s, err: %s\n", folderInfo.Id, folderInfo.Title, err)
	}

	crit = bson.D{{"chatid", bson.M{"$nin": chatFolder.IncludedChatIds}}, {"listid", folderInfo.Id}}
	dr, err := m.chatListColl.DeleteMany(mongoContext, crit)
	if err != nil {
		fmt.Printf("Failed to delete chats from folder id: %d, n: %s, err: %s\n", folderInfo.Id, folderInfo.Title, err)
	} else if dr.DeletedCount > 0 {
		fmt.Printf("Deleted %d chats from folder id: %d, name: %s\n", dr.DeletedCount, folderInfo.Id, folderInfo.Title)
	}
}

func (m *TdMongo) SaveAllChatPositions(chatId int64, positions []*client.ChatPosition) {
	if len(positions) == 0 {
		return
	}
	for _, pos := range positions {
		m.SaveChatPosition(chatId, pos)
	}
}

func (m *TdMongo) SaveChatPosition(chatId int64, chatPosition *client.ChatPosition) {
	var listId int32
	//@TODO: mongo should not be dependent of go-tdlib/client
	clType := chatPosition.List.ChatListType()
	switch clType {
	case "chatListArchive":
		//l := chatPosition.List.(*client.ChatListArchive)
		listId = consts.ClArchive
		break
	case "chatListMain":
		//l := chatPosition.List.(*client.ChatListMain)
		listId = consts.ClMain
		break
	case "chatListFolder":
		l := chatPosition.List.(*client.ChatListFolder)
		listId = l.ChatFolderId
		break
	default:
		listId = consts.ClCached
		fmt.Printf("Invalid chat position type: %s", clType)
	}
	//fmt.Printf("ChatPosition update: %d | %d | %d | %s\n", chatId, chatPosition.Order, listId, chatPosition.List.ChatListType())

	filStr := structs.ChatPosition{ChatId: chatId, Order: int64(chatPosition.Order), IsPinned: chatPosition.IsPinned, ListId: listId}
	crit := bson.D{{"chatid", chatId}, {"listid", listId}}
	update := bson.D{{"$set", filStr}}
	t := true
	opts := &options.UpdateOptions{Upsert: &t}
	_, err := m.chatListColl.UpdateOne(mongoContext, crit, update, opts)
	if err != nil {
		fmt.Printf("Failed to save chatPosition: %d | %d: %s", chatId, chatPosition.Order, err)
	}
}

func (m *TdMongo) GetSavedChats(listId int32) []structs.ChatPosition {
	crit := bson.D{{"listid", listId}}
	opts := options.FindOptions{Sort: bson.M{"order": -1}}
	cur, err := m.chatListColl.Find(mongoContext, crit, &opts)
	var list []structs.ChatPosition
	if err != nil {
		fmt.Printf("Chat list error: %s", err)

		return list
	}
	var chatsMongo []bson.M
	err = cur.All(mongoContext, &chatsMongo)
	if err != nil {
		errmsg := fmt.Sprintf("ERROR mongo select: %s", err)
		fmt.Printf(errmsg)

		return list
	}
	var chats []structs.ChatPosition
	for _, chatObj := range chatsMongo {
		chat := structs.ChatPosition{
			IsPinned: chatObj["ispinned"].(bool),
			Order:    chatObj["order"].(int64),
			ChatId:   chatObj["chatid"].(int64),
		}
		chats = append(chats, chat)
	}

	return chats
}

func (m *TdMongo) ClearChatFilters() {
	removed, err := m.chatFiltersColl.DeleteMany(mongoContext, bson.M{})
	if err != nil {
		log.Printf("Failed to remove chat folders from db: %s", err.Error())
		return
	}
	log.Printf("Removed %d chat folders from db", removed.DeletedCount)
}

func (m *TdMongo) LoadChatFolders() []structs.ChatFilter {
	cur, _ := m.chatFiltersColl.Find(mongoContext, bson.M{})
	fi := make([]structs.ChatFilter, 0)
	err := cur.All(mongoContext, &fi)
	if err != nil {
		errmsg := fmt.Sprintf("ERROR load chat filters: %s", err)
		fmt.Printf(errmsg)

		return fi
	}
	log.Printf("Loaded %d chat folders from db", len(fi))

	return fi
}

func (m *TdMongo) loadSettings() {
	crit := bson.D{{"t", "ignore_lists"}}
	ignoreListsDoc := m.settingsColl.FindOne(mongoContext, crit)
	var il structs.IgnoreLists
	if ignoreListsDoc.Err() == mongo.ErrNoDocuments {
		log.Printf("No ignore lists in DB!")
		il = structs.IgnoreLists{
			T:               "ignore_lists",
			IgnoreAuthorIds: make(map[string]bool),
			IgnoreChatIds:   make(map[string]bool),
			IgnoreFolders:   make(map[string]bool),
		}
		m.settings = il
		return
	}

	err := ignoreListsDoc.Decode(&il)
	if err != nil {
		log.Fatalf("Cannot load ignore lists: %s", err.Error())
	}
	log.Printf("Loaded settings OK!")

	m.settings = il
}

func (m *TdMongo) GetSettings() structs.IgnoreLists {
	//@TODO: how to check empty struct?
	if m.settings.IgnoreAuthorIds == nil {
		m.loadSettings()
	}

	return m.settings
}

func (m *TdMongo) SaveSettings(il structs.IgnoreLists) {
	crit := bson.D{{"t", "ignore_lists"}}
	update := bson.D{{"$set", il}}
	t := true
	opts := &options.UpdateOptions{Upsert: &t}
	_, err := m.settingsColl.UpdateOne(mongoContext, crit, update, opts)
	if err != nil {
		fmt.Printf("Failed to save ignoreLists: %s", err)
	}
}

func (m *TdMongo) DeleteChatFolder(folderId int32) (*mongo.DeleteResult, error) {
	crit := bson.D{{"listid", folderId}}

	return m.chatListColl.DeleteMany(mongoContext, crit)
}
