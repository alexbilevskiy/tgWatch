package helpers

import (
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"tgWatch/config"
	"tgWatch/structs"
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
	var res structs.TgUpdate
	err := msg.Decode(&res)
	if err != nil {
		return fmt.Sprintf("ERROR: %s", err)
	}


	return res.T
}

