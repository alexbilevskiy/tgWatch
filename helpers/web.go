package helpers

import (
	"encoding/json"
	"fmt"
	"go-tdlib/client"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"tgWatch/config"
	"tgWatch/structs"
	"time"
)

var verbose bool = false

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

	m := r.FindStringSubmatch(req.URL.Path)
	if m == nil {
		data := []byte(fmt.Sprintf("Unknown path %s", req.RequestURI))
		res.Write(data)

		return
	}
	data := []byte(fmt.Sprintf("Request URL: %s", req.RequestURI))

	action := m[1]
	req.ParseForm()
	if req.FormValue("a") == "1" {
		verbose = true
	} else {
		verbose = false
	}

	switch action {
	case "e":
		r := regexp.MustCompile(`^/e/(-?\d+)/(\d+)$`)
		m := r.FindStringSubmatch(req.URL.Path)
		if m == nil {
			data := []byte(fmt.Sprintf("Unknown path %s", req.URL.Path))
			res.Write(data)

			return
		}
		chatId, _ := strconv.ParseInt(m[1], 10, 64)
		messageId, _ := strconv.ParseInt(m[2], 10, 64)
		data = []byte(processTgEdit(chatId, messageId))
		break
	case "d":
		r := regexp.MustCompile(`^/d/(-?\d+)/([\d,]+)$`)
		m := r.FindStringSubmatch(req.URL.Path)
		if m == nil {
			data := []byte(fmt.Sprintf("Unknown path %s", req.URL.Path))
			res.Write(data)

			return
		}
		chatId, _ := strconv.ParseInt(m[1], 10, 64)
		messageIds := m[2]
		data = processTgDelete(chatId, ExplodeInt(messageIds))
		break
	}


	res.Write(data)
}

func processTgDelete(chatId int64, messageIds []int64) []byte {

	var fullContentJ []interface{}
	for _, messageId := range messageIds {
		upd, err := FindUpdateNewMessage(messageId)
		if err != nil {
			m := structs.MessageError{T: "Not found deleted", MessageId: messageId, Error: fmt.Sprintf("Error: %s", err)}
			fullContentJ = append(fullContentJ, m)

			continue
		}

		m := parseUpdateNewMessage(upd)
		m.T = "Deleted Message"
		fullContentJ = append(fullContentJ, parseUpdateNewMessage(upd))
	}
	j, _ := json.Marshal(fullContentJ)

	return j
}

func processTgEdit(chatId int64, messageId int64) []byte {
	updates, updateTypes, err := FindAllMessageChanges(messageId)
	if err != nil {
		return []byte("not found messages")
	}

	var fullContentJ []interface{}
	for i, rawJsonBytes := range updates {
		switch updateTypes[i] {
		case "updateNewMessage":
			upd, _ := client.UnmarshalUpdateNewMessage(rawJsonBytes)
			fullContentJ = append(fullContentJ, parseUpdateNewMessage(upd))
			break
		case "updateMessageEdited":
			upd, _ := client.UnmarshalUpdateMessageEdited(rawJsonBytes)
			fullContentJ = append(fullContentJ, parseUpdateMessageEdited(upd))
			break
		case "updateMessageContent":
			upd, _ := client.UnmarshalUpdateMessageContent(rawJsonBytes)
			fullContentJ = append(fullContentJ, parseUpdateMessageContent(upd))
			break
		default:
			fmt.Printf("Invalid update received from mongo: %s", updateTypes[i])
		}
	}
	j, _ := json.Marshal(fullContentJ)

	return j
}

func parseUpdateMessageEdited(upd *client.UpdateMessageEdited) structs.MessageEditedMeta {
	m := structs.MessageEditedMeta{
		T:         "Meta",
		MessageId: upd.MessageId,
		Date:      upd.EditDate,
		DateStr:   time.Unix(int64(upd.EditDate), 0).Format(time.RFC3339),
	}

	return m
}

func parseUpdateNewMessage(upd *client.UpdateNewMessage) structs.MessageInfo {
	content := GetContent(upd.Message.Content)

	senderChatId := GetChatIdBySender(upd.Message.Sender)

	result := structs.MessageInfo{
		T:          "NewMessage",
		MessageId:  upd.Message.ChatId,
		Date:       upd.Message.Date,
		DateStr:    time.Unix(int64(upd.Message.Date), 0).Format(time.RFC3339),
		ChatId:     upd.Message.ChatId,
		ChatName:   GetChatName(upd.Message.ChatId),
		SenderId:   senderChatId,
		SenderName: GetSenderName(upd.Message.Sender),
		Content:    content,
		ContentRaw: nil,
	}
	if verbose {
		result.ContentRaw = upd.Message.Content
	}

	return result
}

func parseUpdateMessageContent(upd *client.UpdateMessageContent) structs.MessageNewContent {
	result := structs.MessageNewContent{
		T:          "NewContent",
		MessageId:  upd.MessageId,
		Content:    GetContent(upd.NewContent),
		ContentRaw: nil,
	}
	if verbose {
		result.ContentRaw = upd.NewContent
	}

	return result
}