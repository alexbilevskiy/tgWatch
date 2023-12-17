package libs

import (
	"context"
	"fmt"
	"github.com/alexbilevskiy/tgWatch/pkg/config"
	"github.com/alexbilevskiy/tgWatch/pkg/structs"
	"github.com/zelenin/go-tdlib/client"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"
)

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

func InitMongo(acc int64) {
	db := Accounts[acc].DbPrefix + Accounts[acc].Phone
	updatesColl[acc] = mongoClient.Database(db).Collection("updates")
	chatFiltersColl[acc] = mongoClient.Database(db).Collection("chatFilters")
	chatListColl[acc] = mongoClient.Database(db).Collection("chatList")
	settingsColl[acc] = mongoClient.Database(db).Collection("settings")
}

func saveChatFolder(acc int64, chatFolder *client.ChatFolder, folderInfo *client.ChatFolderInfo) {

	filStr := structs.ChatFilter{Id: folderInfo.Id, Title: folderInfo.Title, IncludedChats: chatFolder.IncludedChatIds}
	crit := bson.D{{"id", folderInfo.Id}}
	update := bson.D{{"$set", filStr}}
	t := true
	opts := &options.UpdateOptions{Upsert: &t}
	_, err := chatFiltersColl[acc].UpdateOne(mongoContext, crit, update, opts)
	if err != nil {
		fmt.Printf("Failed to save chat filter: id: %d, n: %s, err: %s\n", folderInfo.Id, folderInfo.Title, err)
	}

	crit = bson.D{{"chatid", bson.M{"$nin": chatFolder.IncludedChatIds}}, {"listid", folderInfo.Id}}
	dr, err := chatListColl[acc].DeleteMany(mongoContext, crit)
	if err != nil {
		fmt.Printf("Failed to delete chats from folder id: %d, n: %s, err: %s\n", folderInfo.Id, folderInfo.Title, err)
	} else if dr.DeletedCount > 0 {
		fmt.Printf("Deleted %d chats from folder id: %d, name: %s\n", dr.DeletedCount, folderInfo.Id, folderInfo.Title)
	}
}

const (
	ClCached        int32 = 0
	ClMain          int32 = -1
	ClArchive       int32 = -2
	ClMy            int32 = -3
	ClNotSubscribed int32 = -4
	ClNotAssigned   int32 = -5
)

func saveAllChatPositions(acc int64, chatId int64, positions []*client.ChatPosition) {
	if len(positions) == 0 {
		return
	}
	for _, pos := range positions {
		saveChatPosition(acc, chatId, pos)
	}
}

func saveChatPosition(acc int64, chatId int64, chatPosition *client.ChatPosition) {
	var listId int32
	clType := chatPosition.List.ChatListType()
	switch clType {
	case "chatListArchive":
		//l := chatPosition.List.(*client.ChatListArchive)
		listId = ClArchive
		break
	case "chatListMain":
		//l := chatPosition.List.(*client.ChatListMain)
		listId = ClMain
		break
	case "chatListFolder":
		l := chatPosition.List.(*client.ChatListFolder)
		listId = l.ChatFolderId
		break
	default:
		listId = ClCached
		fmt.Printf("Invalid chat position type: %s", clType)
	}
	DLog(fmt.Sprintf("ChatPosition update: %d | %d | %d | %s\n", chatId, chatPosition.Order, listId, chatPosition.List.ChatListType()))

	filStr := structs.ChatPosition{ChatId: chatId, Order: int64(chatPosition.Order), IsPinned: chatPosition.IsPinned, ListId: listId}
	crit := bson.D{{"chatid", chatId}, {"listid", listId}}
	update := bson.D{{"$set", filStr}}
	t := true
	opts := &options.UpdateOptions{Upsert: &t}
	_, err := chatListColl[acc].UpdateOne(mongoContext, crit, update, opts)
	if err != nil {
		fmt.Printf("Failed to save chatPosition: %d | %d: %s", chatId, chatPosition.Order, err)
	}
}

func getSavedChats(acc int64, listId int32) []structs.ChatPosition {
	crit := bson.D{{"listid", listId}}
	opts := options.FindOptions{Sort: bson.M{"order": -1}}
	cur, err := chatListColl[acc].Find(mongoContext, crit, &opts)
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

func ClearChatFilters(acc int64) {
	removed, err := chatFiltersColl[acc].DeleteMany(mongoContext, bson.M{})
	if err != nil {
		log.Printf("Failed to remove chat folders from db: %s", err.Error())
		return
	}
	log.Printf("Removed %d chat folders from db", removed.DeletedCount)
}

func LoadChatFolders(acc int64) {
	cur, _ := chatFiltersColl[acc].Find(mongoContext, bson.M{})
	fi := make([]structs.ChatFilter, 0)
	err := cur.All(mongoContext, &fi)
	if err != nil {
		errmsg := fmt.Sprintf("ERROR load chat filters: %s", err)
		fmt.Printf(errmsg)

		return
	}
	chatFolders[acc] = fi
	log.Printf("Loaded %d chat folders from db", len(chatFolders[acc]))
}

func LoadSettings(acc int64) {
	crit := bson.D{{"t", "ignore_lists"}}
	ignoreListsDoc := settingsColl[acc].FindOne(mongoContext, crit)
	if ignoreListsDoc.Err() == mongo.ErrNoDocuments {
		log.Printf("No ignore lists in DB!")
		ignoreLists[acc] = structs.IgnoreLists{
			T:               "ignore_lists",
			IgnoreAuthorIds: make(map[string]bool),
			IgnoreChatIds:   make(map[string]bool),
			IgnoreFolders:   make(map[string]bool),
		}

		return
	}
	il := structs.IgnoreLists{}

	err := ignoreListsDoc.Decode(&il)
	if err != nil {
		log.Fatalf("Cannot load ignore lists: %s", err.Error())
	}
	ignoreLists[acc] = il

	log.Printf("Loaded settings OK!")
}

func saveSettings(acc int64) {
	crit := bson.D{{"t", "ignore_lists"}}
	update := bson.D{{"$set", ignoreLists[acc]}}
	t := true
	opts := &options.UpdateOptions{Upsert: &t}
	_, err := settingsColl[acc].UpdateOne(mongoContext, crit, update, opts)
	if err != nil {
		fmt.Printf("Failed to save ignoreLists: %s", err)
	}
}

func GetAccountsFilter(phone *string) bson.M {
	if phone == nil {
		return bson.M{}
	}
	return bson.M{"phone": phone}
}

func LoadAccounts(crit bson.M) {
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
	Accounts = make(map[int64]structs.Account)
	counter := 0
	for _, accObj := range accountsBson {
		counter++
		acc := structs.Account{
			Id:       accObj["id"].(int64),
			Phone:    accObj["phone"].(string),
			DbPrefix: accObj["dbprefix"].(string),
			DataDir:  accObj["datadir"].(string),
			Status:   accObj["status"].(string),
			Username: "",
		}
		Accounts[acc.Id] = acc
	}
	log.Printf("Loaded %d accounts", counter)
}

func SaveAccount(account *structs.Account) {
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

func GetSavedAccount(phone string) *structs.Account {
	var acc *structs.Account

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
