package helpers

import (
	"fmt"
	"go-tdlib/client"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"tgWatch/config"
)

func initWeb() {
	server := &http.Server{
		Addr:    config.Config.WebListen,
		Handler: HttpHandler{},
	}
	go server.ListenAndServe()
}

type HttpHandler struct{}
func (h HttpHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if req.RequestURI == "/favicon.ico" {
		res.WriteHeader(404)
		res.Write([]byte("Not found"))

		return
	}
	log.Printf("HTTP: %s", req.RequestURI)
	r := regexp.MustCompile(`^/(-?\d+)/(\d+)$`)

	m := r.FindStringSubmatch(req.RequestURI)
	if m == nil {
		data := []byte(fmt.Sprintf("Unknown path %s", req.RequestURI))
		res.Write(data)

		return
	}

	data := []byte(fmt.Sprintf("%s, %s, %s", m[0], m[1], m[2]))
	chatId, _ := strconv.ParseInt(m[1], 10, 64)
	messageId, _ := strconv.ParseInt(m[2], 10, 64)
	data = []byte(processTgMessage(chatId, messageId))

	res.Write(data)
}

func processTgMessage(chatId int64, messageId int64) string {
	//msg := updatesColl.FindOne(mongoContext, bson.D{{"t", "updateNewMessage"}, {"upd.message.id", messageId}})
	//upd, err := client.UnmarshalUpdateNewMessage(rawJsonBytes)

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

		return errmsg
	}
	fullContent := ""
	for _, updObj := range updates {
		rawJsonBytes := updObj["raw"].(primitive.Binary).Data
		t := updObj["t"].(string)
		content := ""
		switch t {
		case "updateNewMessage":
			content = singleUpdateNewMessage(rawJsonBytes)
			break
		case "updateMessageEdited":
			content = singleUpdateMessageEdited(rawJsonBytes)
			break
		case "updateMessageContent":
			content = singleUpdateMessageContent(rawJsonBytes)
			break
		default:
			fmt.Printf("Invalid update received from mongo: %s", t)
		}
		fullContent = fmt.Sprintf("%s\n\n%s", fullContent, content)
	}

	return fullContent
}

func singleUpdateMessageEdited(rawJsonBytes []byte) string {
	upd, err := client.UnmarshalUpdateMessageEdited(rawJsonBytes)
	if err != nil {
		fmt.Printf("Error decode update: %s", err)

		return "Failed decode! " + string(rawJsonBytes)
	}

	text := fmt.Sprintf("Edited at %d", upd.EditDate)

	return text
}

func singleUpdateNewMessage(rawJsonBytes []byte) string {
	upd, err := client.UnmarshalUpdateNewMessage(rawJsonBytes)
	if err != nil {
		fmt.Printf("Error decode update: %s", err)

		return "Failed decode! " + string(rawJsonBytes)
	}
	content := GetContent(upd.Message.Content)

	senderChatId := GetChatIdBySender(upd.Message.Sender)

	text := fmt.Sprintf(
		"New Message!\n"+
			"date: %d\n"+
			"Chat ID: %d\n"+
			"Chat name: %s\n"+
			"sender ID: %d\n"+
			"sender name: %s\n"+
			"content: %s",
		upd.Message.Date, upd.Message.ChatId, GetChatName(upd.Message.ChatId), senderChatId, GetSenderName(upd.Message.Sender), content)

	return text
}

func singleUpdateMessageContent(rawJsonBytes []byte) string {
	upd, err := client.UnmarshalUpdateMessageContent(rawJsonBytes)
	if err != nil {
		fmt.Printf("Error decode update: %s", err)

		return "Failed decode! " + string(rawJsonBytes)
	}
	content := GetContent(upd.NewContent)

	text := fmt.Sprintf("New content: %s",content)

	return text
}


