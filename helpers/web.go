package helpers

import (
	"fmt"
	"go-tdlib/client"
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
	r := regexp.MustCompile(`^/([a-z])/.+$`)

	m := r.FindStringSubmatch(req.RequestURI)
	if m == nil {
		data := []byte(fmt.Sprintf("Unknown path %s", req.RequestURI))
		res.Write(data)

		return
	}
	data := []byte(fmt.Sprintf("Request URL: %s", req.RequestURI))

	action := m[1]

	switch action {
	case "e":
		r := regexp.MustCompile(`^/e/(-?\d+)/(\d+)$`)
		m := r.FindStringSubmatch(req.RequestURI)
		if m == nil {
			data := []byte(fmt.Sprintf("Unknown path %s", req.RequestURI))
			res.Write(data)

			return
		}
		chatId, _ := strconv.ParseInt(m[1], 10, 64)
		messageId, _ := strconv.ParseInt(m[2], 10, 64)
		data = []byte(processTgEdit(chatId, messageId))
		break
	case "d":
		r := regexp.MustCompile(`^/d/(-?\d+)/([\d,]+)$`)
		m := r.FindStringSubmatch(req.RequestURI)
		if m == nil {
			data := []byte(fmt.Sprintf("Unknown path %s", req.RequestURI))
			res.Write(data)

			return
		}
		chatId, _ := strconv.ParseInt(m[1], 10, 64)
		messageIds := m[2]
		data = []byte(processTgDelete(chatId, ExplodeInt(messageIds)))
		break
	}


	res.Write(data)
}

func processTgDelete(chatId int64, messageIds []int64) string {

	fullContent := ""
	for _, messageId := range messageIds {
		upd, err := FindUpdateNewMessage(messageId)
		if err != nil {
			fullContent += fmt.Sprintf("\n\nmessage %d failed: %s", messageId, err)

			continue
		}

		fullContent += "\n\n" + fmt.Sprintf("Deleted %d:\n%s", messageId, parseUpdateNewMessage(upd))
	}

	return fullContent
}

func processTgEdit(chatId int64, messageId int64) string {
	updates, updateTypes, err := FindAllMessageChanges(messageId)
	if err != nil {
		return "not found messages"
	}
	fullContent := ""
	for i, rawJsonBytes := range updates {
		content := ""
		switch updateTypes[i] {
		case "updateNewMessage":
			upd, _ := client.UnmarshalUpdateNewMessage(rawJsonBytes)
			content = "New messsage: \n" + parseUpdateNewMessage(upd)
			break
		case "updateMessageEdited":
			upd, _ := client.UnmarshalUpdateMessageEdited(rawJsonBytes)
			content = parseUpdateMessageEdited(upd)
			break
		case "updateMessageContent":
			upd, _ := client.UnmarshalUpdateMessageContent(rawJsonBytes)
			content = parseUpdateMessageContent(upd)
			break
		default:
			fmt.Printf("Invalid update received from mongo: %s", updateTypes[i])
		}
		fullContent += "\n\n" + content
	}

	return fullContent
}

func parseUpdateMessageEdited(upd *client.UpdateMessageEdited) string {
	text := fmt.Sprintf("Edited at %d", upd.EditDate)

	return text
}

func parseUpdateNewMessage(upd *client.UpdateNewMessage) string {
	content := GetContent(upd.Message.Content)

	senderChatId := GetChatIdBySender(upd.Message.Sender)

	text := fmt.Sprintf(
			"date: %d\n"+
			"Chat ID: %d\n"+
			"Chat name: %s\n"+
			"sender ID: %d\n"+
			"sender name: %s\n"+
			"content: %s",
		upd.Message.Date, upd.Message.ChatId, GetChatName(upd.Message.ChatId), senderChatId, GetSenderName(upd.Message.Sender), content)

	return text
}

func parseUpdateMessageContent(upd *client.UpdateMessageContent) string {
	content := GetContent(upd.NewContent)

	text := fmt.Sprintf("New content: %s",content)

	return text
}


