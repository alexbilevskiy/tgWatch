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

func initMongo() {
	mongoContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
	}
	updatesColl = mongoClient.Database(config.Config.Mongo["db"]).Collection("updates")
	chatFiltersColl = mongoClient.Database(config.Config.Mongo["db"]).Collection("chatFilters")
	chatListColl = mongoClient.Database(config.Config.Mongo["db"]).Collection("chatList")
}

func SaveUpdate(t string, upd interface{}, timestamp int32) string {
	if timestamp == 0 {
		timestamp = int32(time.Now().Unix())
	}
	r, err := json.Marshal(upd)
	if err != nil {
		log.Printf("[ERROR] json encode update %s: %s", t, err)
		r = nil
	}

	update := structs.TgUpdate{T: t, Time: timestamp, Upd: upd, Raw: r}
	res, err := updatesColl.InsertOne(mongoContext, update)
	if err != nil {
		log.Printf("[ERROR] insert %s: %s", t, err)

		return ""
	}

	return res.InsertedID.(primitive.ObjectID).String()
}

func FindUpdateNewMessage(chatId int64, messageId int64) (*client.UpdateNewMessage, error) {
	msg := updatesColl.FindOne(mongoContext, bson.D{{"t", "updateNewMessage"}, {"upd.message.id", messageId}, {"upd.message.chatid", chatId}})
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

func FindAllMessageChanges(chatId int64, messageId int64) ([][]byte, []string, []int32, error) {
	crit := bson.D{
		{"$or", []interface{}{
			bson.D{{"t", "updateNewMessage"}, {"upd.message.id", messageId}, {"upd.message.chatid", chatId}},
			bson.D{{"t", "updateMessageEdited"}, {"upd.messageid", messageId}, {"upd.chatid", chatId}},
			bson.D{{"t", "updateMessageContent"}, {"upd.messageid", messageId}, {"upd.chatid", chatId}},
			bson.D{{"t", "updateDeleteMessages"}, {"upd.messageids", messageId}, {"upd.chatid", chatId}},
		}},
	}
	cur, _ := updatesColl.Find(mongoContext, crit)

	return iterateCursor(cur)
}

func FindRecentChanges(limit int64) ([][]byte, []string, []int32, error) {
	availableTypes := []string{
		//"updateNewMessage",
		"updateMessageContent",
		"updateDeleteMessages",
	}
	crit := bson.D{{"t", bson.M{"$in": availableTypes}}}
	lim := &limit
	opts := options.FindOptions{Limit: lim, Sort: bson.M{"_id": -1}}
	cur, _ := updatesColl.Find(mongoContext, crit, &opts)

	return iterateCursor(cur)
}

func GetChatsStats(chats []int64) ([]structs.ChatCounters, error) {
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

	cur, err := updatesColl.Aggregate(mongoContext, agg)
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

func GetChatHistory(chatId int64, limit int64, offset int64) ([][]byte, []string, []int32, error) {
	crit := bson.D{{"t", "updateNewMessage"}, {"upd.message.chatid", chatId}}
	lim := &limit
	offs := &offset
	opts := options.FindOptions{Limit: lim, Skip: offs, Sort: bson.M{"_id": -1}}
	DLog(fmt.Sprintf("History opts: %s", JsonMarshalStr(opts)))
	cur, _ := updatesColl.Find(mongoContext, crit, &opts)

	return iterateCursor(cur)
}

func iterateCursor(cur *mongo.Cursor) ([][]byte, []string, []int32, error) {
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

func SaveChatFilters(chatFilters *client.UpdateChatFilters) {
	fmt.Printf("Chat filters update! %s\n", chatFilters.Type)

	for _, filterInfo := range chatFilters.ChatFilters {
		fmt.Printf("New chat filter: id: %d, n: %s\n", filterInfo.Id, filterInfo.Title)
		//@TODO: tg request logic shoud be in tg.go
		req := &client.GetChatFilterRequest{ChatFilterId: filterInfo.Id}
		chatFilter, err := tdlibClient.GetChatFilter(req)
		if err != nil {
			fmt.Printf("Failed to load chat filter: id: %d, n: %s\n", filterInfo.Id, filterInfo.Title)

			continue
		}
		filStr := structs.ChatFilter{Id: filterInfo.Id, Title: filterInfo.Title, IncludedChats: chatFilter.IncludedChatIds}
		crit := bson.D{{"id", filterInfo.Id}}
		update := bson.D{{"$set", filStr}}
		t := true
		opts := &options.UpdateOptions{Upsert: &t}
		_, err = chatFiltersColl.UpdateOne(mongoContext, crit, update, opts)
		if err != nil {
			fmt.Printf("Failed to save chat filter: id: %d, n: %s, err: %s\n", filterInfo.Id, filterInfo.Title, err)
		}
	}
	LoadChatFilters()
}

const (
	ClDefault int32 = 0
	ClMain int32 = -1
	ClArchive int32 = -2
)

func saveAllChatPositions(chatId int64, positions []*client.ChatPosition) {
	if len(positions) == 0 {
		return
	}
	for _, pos := range positions {
		saveChatPosition(chatId, pos)
	}
}

func saveChatPosition(chatId int64, chatPosition *client.ChatPosition) {
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
		listId = ClDefault
		fmt.Printf("Invalid chat position type: %s", clType)
	}
	DLog(fmt.Sprintf("ChatPosition update: %d | %d | %d | %s\n", chatId, chatPosition.Order, listId, chatPosition.List.ChatListType()))

	filStr := structs.ChatPosition{ChatId: chatId, Order: int64(chatPosition.Order), IsPinned: chatPosition.IsPinned, ListId: listId}
	crit := bson.D{{"chatid", chatId}, {"listid", listId}}
	update := bson.D{{"$set", filStr}}
	t := true
	opts := &options.UpdateOptions{Upsert: &t}
	_, err := chatListColl.UpdateOne(mongoContext, crit, update, opts)
	if err != nil {
		fmt.Printf("Failed to save chatPosition: %d | %d: %s", chatId, chatPosition.Order, err)
	}
}

func getSavedChats(listId int32) []structs.ChatPosition {
	crit := bson.D{{"listid", listId}}
	opts := options.FindOptions{Sort: bson.M{"order": -1}}
	cur, err := chatListColl.Find(mongoContext, crit, &opts)
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

func LoadChatFilters() {
	cur, _ := chatFiltersColl.Find(mongoContext, bson.M{})
	err := cur.All(mongoContext, &chatFilters);
	if err != nil {
		errmsg := fmt.Sprintf("ERROR load chat filters: %s", err)
		fmt.Printf(errmsg)

		return
	}
	log.Printf("Loaded %d chat folders", len(chatFilters))
}