package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/alexbilevskiy/tgWatch/internal/consts"
	"github.com/zelenin/go-tdlib/client"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type TdMongo struct {
	updatesColl     *mongo.Collection
	chatFiltersColl *mongo.Collection
	chatListColl    *mongo.Collection
	settingsColl    *mongo.Collection
}

func NewTdMongo(mongoClient *mongo.Client, DbPrefix string, Phone string) *TdMongo {
	db := DbPrefix + Phone
	m := &TdMongo{}
	m.updatesColl = mongoClient.Database(db).Collection("updates")
	m.chatFiltersColl = mongoClient.Database(db).Collection("chatFilters")
	m.chatListColl = mongoClient.Database(db).Collection("chatList")
	m.settingsColl = mongoClient.Database(db).Collection("settings")

	m.LoadChatFolders()

	return m
}

func (m *TdMongo) SaveChatFolder(chatFolder *client.ChatFolder, folderInfo *client.ChatFolderInfo) {

	filStr := ChatFilter{Id: folderInfo.Id, Title: folderInfo.Name.Text.Text, IncludedChats: chatFolder.IncludedChatIds}
	crit := bson.D{{"id", folderInfo.Id}}
	update := bson.D{{"$set", filStr}}
	t := true
	mctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	opts := &options.UpdateOptions{Upsert: &t}
	_, err := m.chatFiltersColl.UpdateOne(mctx, crit, update, opts)
	if err != nil {
		fmt.Printf("Failed to save chat filter: id: %d, n: %s, err: %s\n", folderInfo.Id, folderInfo.Name.Text.Text, err)
	}

	crit = bson.D{{"chatid", bson.M{"$nin": chatFolder.IncludedChatIds}}, {"listid", folderInfo.Id}}
	dr, err := m.chatListColl.DeleteMany(mctx, crit)
	if err != nil {
		fmt.Printf("Failed to delete chats from folder id: %d, n: %s, err: %s\n", folderInfo.Id, folderInfo.Name.Text.Text, err)
	} else if dr.DeletedCount > 0 {
		fmt.Printf("Deleted %d chats from folder id: %d, name: %s\n", dr.DeletedCount, folderInfo.Id, folderInfo.Name.Text.Text)
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
	clType := chatPosition.List.ChatListConstructor()
	switch clType {
	case client.ConstructorChatListArchive:
		//l := chatPosition.List.(*client.ChatListArchive)
		listId = consts.ClArchive
		break
	case client.ConstructorChatListMain:
		//l := chatPosition.List.(*client.ChatListMain)
		listId = consts.ClMain
		break
	case client.ConstructorChatListFolder:
		l := chatPosition.List.(*client.ChatListFolder)
		listId = l.ChatFolderId
		break
	default:
		listId = consts.ClCached
		fmt.Printf("Invalid chat position type: %s", clType)
	}
	//fmt.Printf("ChatPosition update: %d | %d | %d | %s\n", chatId, chatPosition.Order, listId, chatPosition.List.ChatListType())

	filStr := ChatPosition{ChatId: chatId, Order: int64(chatPosition.Order), IsPinned: chatPosition.IsPinned, ListId: listId}
	crit := bson.D{{"chatid", chatId}, {"listid", listId}}
	update := bson.D{{"$set", filStr}}
	t := true
	mctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	opts := &options.UpdateOptions{Upsert: &t}
	_, err := m.chatListColl.UpdateOne(mctx, crit, update, opts)
	if err != nil {
		fmt.Printf("Failed to save chatPosition: %d | %d: %s", chatId, chatPosition.Order, err)
	}
}

func (m *TdMongo) GetSavedChats(listId int32) []ChatPosition {
	mctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	crit := bson.D{{"listid", listId}}
	opts := options.FindOptions{Sort: bson.M{"order": -1}}
	cur, err := m.chatListColl.Find(mctx, crit, &opts)
	var list []ChatPosition
	if err != nil {
		fmt.Printf("Chat list error: %s", err)

		return list
	}
	var chatsMongo []bson.M
	err = cur.All(mctx, &chatsMongo)
	if err != nil {
		errmsg := fmt.Sprintf("ERROR mongo select: %s", err)
		fmt.Printf(errmsg)

		return list
	}
	var chats []ChatPosition
	for _, chatObj := range chatsMongo {
		chat := ChatPosition{
			IsPinned: chatObj["ispinned"].(bool),
			Order:    chatObj["order"].(int64),
			ChatId:   chatObj["chatid"].(int64),
		}
		chats = append(chats, chat)
	}

	return chats
}

func (m *TdMongo) ClearChatFilters() {
	mctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	removed, err := m.chatFiltersColl.DeleteMany(mctx, bson.M{})
	if err != nil {
		log.Printf("Failed to remove chat folders from db: %s", err.Error())
		return
	}
	log.Printf("Removed %d chat folders from db", removed.DeletedCount)
}

func (m *TdMongo) LoadChatFolders() []ChatFilter {
	mctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cur, _ := m.chatFiltersColl.Find(mctx, bson.M{})
	fi := make([]ChatFilter, 0)
	err := cur.All(mctx, &fi)
	if err != nil {
		errmsg := fmt.Sprintf("ERROR load chat filters: %s", err)
		fmt.Printf(errmsg)

		return fi
	}
	log.Printf("Loaded %d chat folders from db", len(fi))

	return fi
}

func (m *TdMongo) DeleteChatFolder(folderId int32) (int64, error) {
	mctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	crit := bson.D{{"listid", folderId}}
	d, err := m.chatListColl.DeleteMany(mctx, crit)

	return d.DeletedCount, err
}
