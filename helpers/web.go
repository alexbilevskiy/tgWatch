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

	msg := updatesColl.FindOne(mongoContext, bson.D{{"t", "updateNewMessage"}, {"upd.message.id", messageId}})

	var res bson.M
	err := msg.Decode(&res)
	if err != nil {
		errmsg := fmt.Sprintf("ERROR mongo decode: %s", err)
		fmt.Printf(errmsg)

		return errmsg
	}
	rawJsonBytes := res["raw"].(primitive.Binary).Data

	fmt.Printf("RAW JSON: %s\n", string(rawJsonBytes))
	upd, err := client.UnmarshalUpdateNewMessage(rawJsonBytes)
	if err != nil {
		fmt.Printf("Error decode update: %s", err)

		return "Failed decode! " + string(rawJsonBytes)
	}
	content := GetContent(upd.Message.Content)

	senderChatId := getChatIdBySender(upd.Message.Sender)

	text := fmt.Sprintf("sender id: %d\nsender name: %s\ncontent: %s", senderChatId, GetChatName(senderChatId), content)

	return text
}


