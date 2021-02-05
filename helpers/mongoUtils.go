package helpers

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
	clientOptions := options.Client().ApplyURI(config.Config.MongoUri).SetRegistry(registry)

	var err error
	mongoClient, err = mongo.Connect(mongoContext, clientOptions)
	if err != nil {
		log.Fatalf("Mongo error: %s", err)
	}
	updatesColl = mongoClient.Database("tg").Collection("updates")
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

func FindUpdateNewMessage(messageId int64) (*client.UpdateNewMessage, error) {
	msg := updatesColl.FindOne(mongoContext, bson.D{{"t", "updateNewMessage"}, {"upd.message.id", messageId}})
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

func FindAllMessageChanges(messageId int64) ([][]byte, []string, error) {
	crit := bson.D{
		{"$or", []interface{}{
			bson.D{{"t", "updateNewMessage"}, {"upd.message.id", messageId}},
			bson.D{{"t", "updateMessageEdited"}, {"upd.messageid", messageId}},
			bson.D{{"t", "updateMessageContent"}, {"upd.messageid", messageId}},
		}},
	}
	cur, _ := updatesColl.Find(mongoContext, crit)

	var updates []bson.M
	err := cur.All(mongoContext, &updates);
	if err != nil {
		errmsg := fmt.Sprintf("ERROR mongo select: %s", err)
		fmt.Printf(errmsg)

		return nil, nil, errors.New("failed mongo select")
	}
	var jsons [][]byte
	var types []string
	for _, updObj := range updates {
		rawJsonBytes := updObj["raw"].(primitive.Binary).Data
		t := updObj["t"].(string)
		types = append(types, t)
		jsons = append(jsons, rawJsonBytes)
	}

	return jsons, types, nil
}