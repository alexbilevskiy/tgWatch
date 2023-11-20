package libs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/alexbilevskiy/tgWatch/pkg/config"
	"github.com/alexbilevskiy/tgWatch/pkg/structs"
	"github.com/zelenin/go-tdlib/client"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"reflect"
	"time"
)

//@TODO: this is an attempt to make direct unmarshall from mongo BSON to telegram structs, but it doesn't work.
//Luckily though, unmarshalling from JSON works fine, so as temporary solution we just store raw JSON in mongo and unmarshal it instead.
type messageSenderDecoder struct{}

func (n messageSenderDecoder) DecodeValue(decodeContext bsoncodec.DecodeContext, reader bsonrw.ValueReader, value reflect.Value) error {

	fmt.Printf("UPD DEC TYPE: %s, %s\n", reader.Type(), value.Type())
	doc, err := reader.ReadDocument()
	if err != nil {
		fmt.Printf("UPD DEC error %s\n", err)

		return nil
	}
	elem, vr, err := doc.ReadElement()
	if err != nil {
		fmt.Printf("UPD ELEM error %s\n", err)

		return nil
	}
	if elem == "chatid" {
		fmt.Printf("UPD DEC CHATID: %s, %s\n", elem, vr.Type())
		chatId, err := vr.ReadInt64()
		if err != nil {
			fmt.Printf("UPD ELEM CHAT error %s\n", err)

			return nil
		}
		fmt.Printf("T: %s, %s\n", value.Type(), value.Interface())

		//ms := client.MessageSenderChat{ChatId: chatId}
		_ = client.MessageSenderChat{ChatId: chatId}

		//var a client.MessageSender
		//newV := reflect.ValueOf(a)

		//fmt.Print("CS1:", value.CanSet(), "\n" )
		//fmt.Print("CS2:", value.Addr().CanSet(),  "\n")
		//fmt.Print("CS3:", value.Addr().Elem().CanSet(),  "\n")
		//fmt.Print("CS4:", reflect.ValueOf(&ms).CanAddr(), "\n")

		//fmt.Print("CS3:", reflect.ValueOf(&ms).Elem().CanAddr(), "\n")
		//value.Addr().Elem().Set(reflect.ValueOf(&ms).Elem())
		//value.Set(reflect.ValueOf(ms.(client.MessageSender)))

	} else {
		fmt.Printf("UPD DEC ELEM: %s, %s\n", elem, vr.Type())
	}

	return nil
}

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

func SaveUpdate(acc int64, t string, upd interface{}, timestamp int32) string {
	if timestamp == 0 {
		timestamp = int32(time.Now().Unix())
	}
	r, err := json.Marshal(upd)
	if err != nil {
		log.Printf("[ERROR] json encode update %s: %s", t, err)
		r = nil
	}

	update := structs.TgUpdate{T: t, Time: timestamp, Upd: upd, Raw: r}
	res, err := updatesColl[acc].InsertOne(mongoContext, update)
	if err != nil {
		log.Printf("[ERROR] insert %s: %s", t, err)

		return ""
	}

	return res.InsertedID.(primitive.ObjectID).String()
}

func FindUpdateNewMessage(acc int64, chatId int64, messageId int64) (*client.UpdateNewMessage, error) {
	msg := updatesColl[acc].FindOne(mongoContext, bson.D{{"t", "updateNewMessage"}, {"upd.message.id", messageId}, {"upd.message.chatid", chatId}})
	if msg == nil {

		return nil, errors.New("message not found")

	}
	var updObj bson.M
	err := msg.Decode(&updObj)
	if err != nil {

		return nil, err
	}
	var rawJsonBytes []byte
	if reflect.TypeOf(updObj["raw"]) == reflect.TypeOf(primitive.Binary{}) {
		rawJsonBytes = updObj["raw"].(primitive.Binary).Data
	} else {
		rawJsonBytes = []byte(updObj["raw"].(string))
	}

	upd, err := client.UnmarshalUpdateNewMessage(rawJsonBytes)
	if err != nil {
		fmt.Printf("Error decode update: %s", err)

		return nil, errors.New("failed to unmarshal update")
	}

	return upd, nil
}

func FindAllMessageChanges(acc int64, chatId int64, messageId int64) ([][]byte, []string, []int32, error) {
	crit := bson.D{
		{"$or", []interface{}{
			bson.D{{"t", "updateNewMessage"}, {"upd.message.id", messageId}, {"upd.message.chatid", chatId}},
			bson.D{{"t", "updateMessageEdited"}, {"upd.messageid", messageId}, {"upd.chatid", chatId}},
			bson.D{{"t", "updateMessageContent"}, {"upd.messageid", messageId}, {"upd.chatid", chatId}},
			bson.D{{"t", "updateDeleteMessages"}, {"upd.messageids", messageId}, {"upd.chatid", chatId}},
		}},
	}
	cur, _ := updatesColl[acc].Find(mongoContext, crit)

	return iterateCursor(acc, cur)
}

func MarkAsDeleted(acc int64, chatId int64, messageIds []int64) {
	crit := bson.D{{"t", "updateNewMessage"}, {"upd.message.id", bson.M{"$in": messageIds}}, {"upd.message.chatid", chatId}}
	update := bson.D{{"$set", bson.M{"deleted": true}}}
	_, err := updatesColl[acc].UpdateMany(mongoContext, crit, update)
	if err != nil {
		fmt.Printf("Failed to update deleted: %d, %s, %s", chatId, JsonMarshalStr(messageIds), err)
	}
}

func IsMessageEdited(acc int64, chatId int64, messageId int64) bool {
	crit := bson.D{{"t", "updateMessageEdited"}, {"upd.messageid", messageId}, {"upd.chatid", chatId}}

	return countBy(acc, crit) > 0
}

func IsMessageDeleted(acc int64, chatId int64, messageId int64) bool {
	crit := bson.D{{"t", "updateDeleteMessages"}, {"upd.messageids", messageId}, {"upd.chatid", chatId}}

	return countBy(acc, crit) > 0
}

func countBy(acc int64, crit bson.D) int64 {
	count, err := updatesColl[acc].CountDocuments(mongoContext, crit)
	if err != nil {
		fmt.Printf("Failed to count edits for %s: %s", JsonMarshalStr(crit), err)

		return -1
	}

	return count
}

func FindRecentChanges(acc int64) (*mongo.Cursor, error) {
	availableTypes := []string{
		//"updateNewMessage",
		//"updateMessageContent",
		"updateDeleteMessages",
	}
	crit := bson.D{{"t", bson.M{"$in": availableTypes}}}
	opts := options.FindOptions{Sort: bson.M{"_id": -1}, Hint: "_id_-1_t_1"}

	return updatesColl[acc].Find(mongoContext, crit, &opts)
}

func GetChatsStats(acc int64, chats []int64) ([]structs.ChatCounters, error) {
	basicCrit := bson.D{{
		"t", bson.D{
			{"$in", bson.A{
				"updateNewMessage",
				"updateMessageEdited",
				"updateDeleteMessages",
			}},
		},
	}}
	var matchRules bson.D
	if len(chats) > 0 {
		chatsCritList := bson.A{}
		for _, chatId := range chats {
			chatsCritList = append(chatsCritList, chatId)
		}
		chatsCrit := bson.D{
			{"$in", chatsCritList},
		}
		chatRules := bson.D{
			{"$or", []interface{}{
				bson.D{{"t", "updateNewMessage"}, {"upd.message.chatid", chatsCrit}},
				bson.D{{"t", "updateMessageEdited"}, {"upd.chatid", chatsCrit}},
				bson.D{{"t", "updateDeleteMessages"}, {"upd.chatid", chatsCrit}},
			}},
		}
		matchRules = bson.D{
			{"$and", []interface{}{
				basicCrit,
				chatRules,
			}},
		}
	} else {
		matchRules = basicCrit
	}

	match := bson.D{
		{"$match", matchRules},
	}
	group := bson.D{
		{"$group", bson.D{
			{"_id", bson.D{{"$cond", bson.A{"$upd.message.chatid", "$upd.message.chatid", bson.D{{"$cond", bson.A{"$upd.chatid", "$upd.chatid", 0}}}}}}},
			{"countUpdateNewMessage", bson.D{{"$sum", bson.D{{"$cond", bson.A{bson.D{{"$eq", bson.A{"$t", "updateNewMessage"}}}, 1, 0}}}}}},
			{"countUpdateMessageEdited", bson.D{{"$sum", bson.D{{"$cond", bson.A{bson.D{{"$eq", bson.A{"$t", "updateMessageEdited"}}}, 1, 0}}}}}},
			{"countUpdateDeleteMessages", bson.D{{"$sum", bson.D{{"$cond", bson.A{bson.D{{"$eq", bson.A{"$t", "updateDeleteMessages"}}}, 1, 0}}}}}},
			{"count", bson.D{{"$sum", 1}}},
		},
		},
	}
	sort := bson.D{
		{"$sort", bson.D{{"count", -1}}},
	}
	DLog(fmt.Sprintf("ChatStats crit: %s", JsonMarshalStr(match)))
	agg := bson.A{match, group, sort}

	cur, err := updatesColl[acc].Aggregate(mongoContext, agg)
	if err != nil {
		errmsg := fmt.Sprintf("ERROR mongo agg: %s\n", err)
		fmt.Printf(errmsg)
		return nil, err
	}
	var chatsStats []bson.M
	err = cur.All(mongoContext, &chatsStats)
	if err != nil {
		errmsg := fmt.Sprintf("ERROR mongo itreate: %s\n", err)
		fmt.Printf(errmsg)

		return nil, errors.New("failed mongo select")
	}
	var result []structs.ChatCounters
	for _, aggItem := range chatsStats {
		c := structs.ChatCounters{
			ChatId: aggItem["_id"].(int64),
		}
		c.Counters = make(map[string]int32, 3)
		c.Counters["total"] = aggItem["count"].(int32)
		c.Counters["updateNewMessage"] = aggItem["countUpdateNewMessage"].(int32)
		c.Counters["updateMessageEdited"] = aggItem["countUpdateMessageEdited"].(int32)
		c.Counters["updateDeleteMessages"] = aggItem["countUpdateDeleteMessages"].(int32)
		result = append(result, c)
	}

	return result, nil
}

func GetChatHistory(acc int64, chatId int64, limit int64, offset int64, deleted bool) ([][]byte, []string, []int32, error) {
	var crit bson.D
	if !deleted {
		crit = bson.D{{"t", "updateNewMessage"}, {"upd.message.chatid", chatId}}
	} else {
		crit = bson.D{{"t", "updateNewMessage"}, {"upd.message.chatid", chatId}, {"deleted", true}}
	}
	lim := &limit
	offs := &offset
	opts := options.FindOptions{Limit: lim, Skip: offs, Sort: bson.M{"_id": -1}}
	DLog(fmt.Sprintf("History opts: %s", JsonMarshalStr(opts)))
	cur, _ := updatesColl[acc].Find(mongoContext, crit, &opts)

	return iterateCursor(acc, cur)
}

func iterateCursor(acc int64, cur *mongo.Cursor) ([][]byte, []string, []int32, error) {
	var updates []bson.M
	err := cur.All(mongoContext, &updates)
	if err != nil {
		errmsg := fmt.Sprintf("ERROR mongo select: %s", err)
		fmt.Printf(errmsg)

		return nil, nil, nil, errors.New("failed mongo select")
	}
	var jsons [][]byte
	var types []string
	var dates []int32
	for _, updObj := range updates {
		rawJsonBytes := updObj["raw"].(primitive.Binary).Data
		t := updObj["t"].(string)
		types = append(types, t)
		jsons = append(jsons, rawJsonBytes)
		dates = append(dates, updObj["time"].(int32))
	}

	return jsons, types, dates, nil
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
