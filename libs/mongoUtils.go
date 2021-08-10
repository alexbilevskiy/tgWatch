package libs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-tdlib/client"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"reflect"
	"tgWatch/config"
	"tgWatch/structs"
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

func InitMongo(acc int32) {
	db := Accounts[acc].DbPrefix + Accounts[acc].Phone
	updatesColl[acc] = mongoClient.Database(db).Collection("updates")
	chatFiltersColl[acc] = mongoClient.Database(db).Collection("chatFilters")
	chatListColl[acc] = mongoClient.Database(db).Collection("chatList")
	settingsColl[acc] = mongoClient.Database(db).Collection("settings")
}

func SaveUpdate(acc int32, t string, upd interface{}, timestamp int32) string {
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

func FindUpdateNewMessage(acc int32, chatId int64, messageId int64) (*client.UpdateNewMessage, error) {
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

func FindAllMessageChanges(acc int32, chatId int64, messageId int64) ([][]byte, []string, []int32, error) {
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

func MarkAsDeleted(acc int32, chatId int64, messageIds []int64) {
	crit := bson.D{{"t", "updateNewMessage"}, {"upd.message.id", bson.M{"$in": messageIds}}, {"upd.message.chatid", chatId}}
	update := bson.D{{"$set", bson.M{"deleted": true}}}
	_, err := updatesColl[acc].UpdateMany(mongoContext, crit, update)
	if err != nil {
		fmt.Printf("Failed to update deleted: %d, %s, %s", chatId, JsonMarshalStr(messageIds), err)
	}
}

func IsMessageEdited(acc int32, chatId int64, messageId int64) bool {
	crit := bson.D{{"t", "updateMessageEdited"}, {"upd.messageid", messageId}, {"upd.chatid", chatId}}

	return countBy(acc, crit) > 0
}

func IsMessageDeleted(acc int32, chatId int64, messageId int64) bool {
	crit := bson.D{{"t", "updateDeleteMessages"}, {"upd.messageids", messageId}, {"upd.chatid", chatId}}

	return countBy(acc, crit) > 0
}

func countBy(acc int32, crit bson.D) int64 {
	count, err := updatesColl[acc].CountDocuments(mongoContext, crit)
	if err != nil {
		fmt.Printf("Failed to count edits for %s: %s", JsonMarshalStr(crit), err)

		return -1
	}

	return count
}

func FindRecentChanges(acc int32, limit int64) ([][]byte, []string, []int32, error) {
	availableTypes := []string{
		//"updateNewMessage",
		"updateMessageContent",
		"updateDeleteMessages",
	}
	crit := bson.D{{"t", bson.M{"$in": availableTypes}}}
	lim := &limit
	opts := options.FindOptions{Limit: lim, Sort: bson.M{"_id": -1}}
	cur, _ := updatesColl[acc].Find(mongoContext, crit, &opts)

	return iterateCursor(acc, cur)
}

func GetChatsStats(acc int32, chats []int64) ([]structs.ChatCounters, error) {
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

func GetChatHistory(acc int32, chatId int64, limit int64, offset int64, deleted bool) ([][]byte, []string, []int32, error) {
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

func iterateCursor(acc int32, cur *mongo.Cursor) ([][]byte, []string, []int32, error) {
	var updates []bson.M
	err := cur.All(mongoContext, &updates);
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

func SaveChatFilters(acc int32, chatFilters *client.UpdateChatFilters) {
	fmt.Printf("Chat filters update! %s\n", chatFilters.Type)

	for _, filterInfo := range chatFilters.ChatFilters {
		fmt.Printf("New chat filter: id: %d, n: %s\n", filterInfo.Id, filterInfo.Title)
		//@TODO: tg request logic shoud be in tg.go
		req := &client.GetChatFilterRequest{ChatFilterId: filterInfo.Id}
		chatFilter, err := tdlibClient[acc].GetChatFilter(req)
		if err != nil {
			fmt.Printf("Failed to load chat filter: id: %d, n: %s\n", filterInfo.Id, filterInfo.Title)

			continue
		}
		filStr := structs.ChatFilter{Id: filterInfo.Id, Title: filterInfo.Title, IncludedChats: chatFilter.IncludedChatIds}
		crit := bson.D{{"id", filterInfo.Id}}
		update := bson.D{{"$set", filStr}}
		t := true
		opts := &options.UpdateOptions{Upsert: &t}
		_, err = chatFiltersColl[acc].UpdateOne(mongoContext, crit, update, opts)
		if err != nil {
			fmt.Printf("Failed to save chat filter: id: %d, n: %s, err: %s\n", filterInfo.Id, filterInfo.Title, err)
		}

		crit = bson.D{{"chatid", bson.M{"$nin": chatFilter.IncludedChatIds}}, {"listid", filterInfo.Id}}
		dr, err := chatListColl[acc].DeleteMany(mongoContext, crit)
		if err != nil {
			fmt.Printf("Failed to delete non-matching chats for filter: id: %d, n: %s, err: %s\n", filterInfo.Id, filterInfo.Title, err)
		} else {
			fmt.Printf("Deleted %d non matching chats, id: %d, name: %s\n", dr.DeletedCount, filterInfo.Id, filterInfo.Title)
		}
	}
	LoadChatFilters(acc)
}

const (
	ClCached        int32 = 0
	ClMain          int32 = -1
	ClArchive       int32 = -2
	ClMy            int32 = -3
	ClNotSubscribed int32 = -4
)

func saveAllChatPositions(acc int32, chatId int64, positions []*client.ChatPosition) {
	if len(positions) == 0 {
		return
	}
	for _, pos := range positions {
		saveChatPosition(acc, chatId, pos)
	}
}

func saveChatPosition(acc int32, chatId int64, chatPosition *client.ChatPosition) {
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
	case "chatListFilter":
		l := chatPosition.List.(*client.ChatListFilter)
		listId = l.ChatFilterId
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

func getSavedChats(acc int32, listId int32) []structs.ChatPosition {
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

func LoadChatFilters(acc int32) {
	cur, _ := chatFiltersColl[acc].Find(mongoContext, bson.M{})
	fi := make([]structs.ChatFilter, 0)
	err := cur.All(mongoContext, &fi)
	if err != nil {
		errmsg := fmt.Sprintf("ERROR load chat filters: %s", err)
		fmt.Printf(errmsg)

		return
	}
	chatFilters[acc] = fi
	log.Printf("Loaded %d chat folders", len(chatFilters[acc]))
}

func LoadSettings(acc int32) {
	crit := bson.D{{"t", "ignore_lists"}}
	ignoreListsDoc := settingsColl[acc].FindOne(mongoContext, crit)
	if ignoreListsDoc.Err() == mongo.ErrNoDocuments {
		log.Printf("No ignore lists in DB!")
		ignoreLists[acc] = structs.IgnoreLists{
			T: "ignore_lists",
			IgnoreAuthorIds: make(map[string]bool),
			IgnoreChatIds: make(map[string]bool),
			IgnoreFolders: make(map[string]bool),
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

func saveSettings(acc int32) {
	crit := bson.D{{"t", "ignore_lists"}}
	update := bson.D{{"$set", ignoreLists[acc]}}
	t := true
	opts := &options.UpdateOptions{Upsert: &t}
	_, err := settingsColl[acc].UpdateOne(mongoContext, crit, update, opts)
	if err != nil {
		fmt.Printf("Failed to save ignoreLists: %s", err)
	}
}

func LoadAccounts() {
	accountsCursor, err := accountColl.Find(mongoContext, bson.M{})
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
	Accounts = make(map[int32]structs.Account)
	counter := 0
	for _, accObj := range accountsBson {
		counter++
		acc := structs.Account{
			Id: accObj["id"].(int32),
			Phone: accObj["phone"].(string),
			DbPrefix: accObj["dbprefix"].(string),
			DataDir: accObj["datadir"].(string),
			Status: accObj["status"].(string),
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