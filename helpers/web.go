package helpers

import (
	"encoding/json"
	"fmt"
	"go-tdlib/client"
	"html/template"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"tgWatch/config"
	"tgWatch/structs"
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
	if req.URL.Path == "/" {
		req.URL.Path = "index.html"
	}
	path := "web/" + req.URL.Path
	stat, err := os.Stat(path);
	if err == nil && !stat.IsDir() {
		http.ServeFile(res, req, path)

		return
	}

	log.Printf("HTTP: %s", req.URL.Path)
	r := regexp.MustCompile(`^/([a-z]+?)($|/.+$)`)

	m := r.FindStringSubmatch(req.URL.Path)
	if m == nil {
		res.WriteHeader(404)
		res.Write([]byte("not found "+ req.URL.Path))

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
	case "routes":

		res.Write([]byte(`{"routes":[{"/j":"journal"}]}`))

		return
	case "e":
		r := regexp.MustCompile(`^/e/(-?\d+)/(\d+)$`)
		m := r.FindStringSubmatch(req.URL.Path)
		if m == nil {
			data := []byte(fmt.Sprintf("Unknown path %s %s", action, req.URL.Path))
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
			data := []byte(fmt.Sprintf("Unknown path %s %s", action, req.URL.Path))
			res.Write(data)

			return
		}
		chatId, _ := strconv.ParseInt(m[1], 10, 64)
		messageIds := m[2]
		data = processTgDelete(chatId, ExplodeInt(messageIds))
		break
	case "j":
		limit := int64(50)
		if req.FormValue("limit") != "" {
			limit, _ = strconv.ParseInt(req.FormValue("limit"), 10, 64)
		}
		processTgJournal(limit, res)
		return
	case "o":
		limit := int64(50)
		if req.FormValue("limit") != "" {
			limit, _ = strconv.ParseInt(req.FormValue("limit"), 10, 64)
		}
		data = processTgOverview(limit)
		break
	case "c":
		r := regexp.MustCompile(`^/c/(-?\d+)$`)
		m := r.FindStringSubmatch(req.URL.Path)
		if m == nil {
			data := []byte(fmt.Sprintf("Unknown path %s %s", action, req.URL.Path))
			res.Write(data)

			return
		}
		chatId, _ := strconv.ParseInt(m[1], 10, 64)
		data = processTgChat(chatId)
		break
	case "f":
		r := regexp.MustCompile(`^/f/((\d+)|([\w\-_]+))$`)
		m := r.FindStringSubmatch(req.URL.Path)
		var file *client.File
		var err error
		if m == nil {
			data := []byte(fmt.Sprintf("Unknown path %s %s", action, req.URL.Path))
			res.Write(data)

			return
		} else if m[2] != "" {
			imageId, _ := strconv.ParseInt(m[2], 10, 32)
			file, err = DownloadFile(int32(imageId))
		} else if m[3] != "" {
			file, err = DownloadFileByRemoteId(m[3])
		} else {
			data := []byte(fmt.Sprintf("Unknown file name %s %s", action, req.URL.Path))
			res.Write(data)

			return
		}
		if err != nil {
			errMsg := structs.MessageAttachmentError{T:"attachmentError", Id: m[1], Error: err.Error()}
			j, _ := json.Marshal(errMsg)
			data = j

			break
		}
		if file.Local.Path != "" && !verbose {
			//res.Header().Add("Content-Type", "file/jpeg")
			http.ServeFile(res, req, file.Local.Path)

			return
		}
		j, _ := json.Marshal(file)
		data = j

		break
	default:
		res.WriteHeader(404)
		res.Write([]byte("not found " + req.URL.Path))

		return
	}

	res.WriteHeader(200)
	res.Write(data)
}

type ChatInfo struct {
	ChatId int64
	ChatName string
}
type JournalItem struct {
	T string
	Time int32
	Date string
	MessageId []int64
	Chat ChatInfo
	From ChatInfo
	Link string
	IntLink string
	Message string
	Error string
}
type Journal struct {
	J []JournalItem
}
type JSON struct {
	JSON string
}
func processTgJournal(limit int64, w http.ResponseWriter)  {
	updates, updateTypes, dates, errSelect := FindRecentChanges(limit)
	if errSelect != nil {
		fmt.Printf("Error select updates: %s\n", errSelect)

		return
	}
	var t *template.Template
	var errParse error
	if verbose {
		t, errParse = template.New(`json.html`).ParseFiles(`templates/json.html`)
	} else {
		t, errParse = template.New(`journal.html`).ParseFiles(`templates/journal.html`)
	}

	if errParse != nil {
		fmt.Printf("Error parse tpl: %s\n", errParse)
		return
	}
	var data Journal

	for i, rawJsonBytes := range updates {
		switch updateTypes[i] {
		case "updateNewMessage":
			upd, _ := client.UnmarshalUpdateNewMessage(rawJsonBytes)
			item := JournalItem{
				T: updateTypes[i],
				Time: dates[i],
				Date: FormatTime(dates[i]),
				Link: GetLink(upd.Message.ChatId, upd.Message.Id),
				IntLink: fmt.Sprintf("/e/%d/%d", upd.Message.ChatId, upd.Message.Id), //@TODO: link shoud be /m
				Chat: ChatInfo{
					ChatId: upd.Message.ChatId,
					ChatName: GetChatName(upd.Message.ChatId),
				},
			}
			if upd.Message.Sender.MessageSenderType() == "messageSenderChat" {
			} else {
				item.From = ChatInfo{ChatId: GetChatIdBySender(upd.Message.Sender), ChatName: GetSenderName(upd.Message.Sender)}
			}
			data.J = append(data.J, item)

			break
		case "updateMessageEdited":
			upd, _ := client.UnmarshalUpdateMessageEdited(rawJsonBytes)
			item := JournalItem{
				T: updateTypes[i],
				Time: dates[i],
				Date: FormatTime(dates[i]),
				Link: GetLink(upd.ChatId, upd.MessageId),
				IntLink: fmt.Sprintf("/e/%d/%d", upd.ChatId, upd.MessageId),
				Chat: ChatInfo{
					ChatId: upd.ChatId,
					ChatName: GetChatName(upd.ChatId),
				},
			}
			data.J = append(data.J, item)

			break
		case "updateMessageContent":
			upd, _ := client.UnmarshalUpdateMessageContent(rawJsonBytes)
			item := JournalItem{
				T: updateTypes[i],
				Time: dates[i],
				Date: FormatTime(dates[i]),
				Link: GetLink(upd.ChatId, upd.MessageId),
				IntLink: fmt.Sprintf("/e/%d/%d", upd.ChatId, upd.MessageId),
				Chat: ChatInfo{
					ChatId: upd.ChatId,
					ChatName: GetChatName(upd.ChatId),
				},
			}
			m, err := FindUpdateNewMessage(upd.MessageId)
			if err != nil {
				item.Error = fmt.Sprintf("Message not found: %s", err)
				data.J = append(data.J, item)

				break
			}

			if m.Message.Sender.MessageSenderType() == "messageSenderChat" {
			} else {
				item.From = ChatInfo{ChatId: GetChatIdBySender(m.Message.Sender), ChatName: GetSenderName(m.Message.Sender)}
			}
			data.J = append(data.J, item)

			break
		case "updateDeleteMessages":
			upd, _ := client.UnmarshalUpdateDeleteMessages(rawJsonBytes)
			item := JournalItem{
				T: updateTypes[i],
				Time: dates[i],
				Date: FormatTime(dates[i]),
				IntLink: fmt.Sprintf("/d/%d/%s", upd.ChatId, ImplodeInt(upd.MessageIds)),
				Chat: ChatInfo{
					ChatId: upd.ChatId,
					ChatName: GetChatName(upd.ChatId),
				},
				MessageId: upd.MessageIds,
			}
			data.J = append(data.J, item)
			break
		default:
			//fc += fmt.Sprintf("[%s] Unknown update type \"%s\"<br>", FormatTime(dates[i]), updateTypes[i])
		}
	}
	var err error
	if verbose {
		err = t.Execute(w, JSON{JSON: JsonMarshalStr(data)})
	} else {
		err = t.Execute(w, data)
	}

	if err != nil {
		fmt.Printf("Error tpl: %s\n", err)
		return
	}
}

func processTgOverview(limit int64) []byte {
	s, err := GetChatsStats()
	if err != nil {

		return []byte(err.Error())
	}
	fc := "<html><body>"
	for _, ci := range s {
		name := GetChatName(ci.ChatId)
		fc += fmt.Sprintf(`<a href="/c/%d">%s</a> (%d total, %d updates, %d deletes)<br>`, ci.ChatId, name, ci.Counters["total"], ci.Counters["updateMesageEdited"], ci.Counters["updateDeleteMessages"])
	}
	fc += "</body></html>"

	return []byte(fc)
}

func processTgDelete(chatId int64, messageIds []int64) []byte {

	var fullContentJ []interface{}
	for _, messageId := range messageIds {
		upd, err := FindUpdateNewMessage(messageId)
		if err != nil {
			m := structs.MessageError{T: "Error", MessageId: messageId, Error: fmt.Sprintf("Error: %s", err)}
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
	var fullContentJ []interface{}

	updates, updateTypes, dates, err := FindAllMessageChanges(messageId)
	if err != nil {
		m := structs.MessageError{T: "Error", MessageId: messageId, Error: fmt.Sprintf("Error: %s", err)}
		fullContentJ = append(fullContentJ, m)
		j, _ := json.Marshal(fullContentJ)

		return j
	}

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
		case "updateDeleteMessages":
			upd, _ := client.UnmarshalUpdateDeleteMessages(rawJsonBytes)
			fullContentJ = append(fullContentJ, parseUpdateDeleteMessages(upd, dates[i]))
			break
		default:
			m := structs.MessageError{T:"Error", MessageId: messageId, Error: fmt.Sprintf("Unknown update type: %s", updateTypes[i])}
			fullContentJ = append(fullContentJ, m)
		}
	}
	j, _ := json.Marshal(fullContentJ)

	return j
}

func processTgChat(chatId int64) []byte {
	var chat interface{}
	var err error
	if chatId > 0 {
		chat, err = GetUser(int32(chatId))
	} else{
		chat, err = GetChat(chatId)
	}
	if err != nil {

		return []byte("Error: " + err.Error())
	}
	j, _ := json.Marshal(chat)

	return j
}

func parseUpdateMessageEdited(upd *client.UpdateMessageEdited) structs.MessageEditedMeta {
	m := structs.MessageEditedMeta{
		T:         "EditedMeta",
		MessageId: upd.MessageId,
		Date:      upd.EditDate,
		DateStr:   FormatTime(upd.EditDate),
	}

	return m
}

func parseUpdateNewMessage(upd *client.UpdateNewMessage) structs.MessageInfo {
	content := GetContent(upd.Message.Content)

	senderChatId := GetChatIdBySender(upd.Message.Sender)

	result := structs.MessageInfo{
		T:           "NewMessage",
		MessageId:   upd.Message.Id,
		Date:        upd.Message.Date,
		DateStr:     FormatTime(upd.Message.Date),
		ChatId:      upd.Message.ChatId,
		ChatName:    GetChatName(upd.Message.ChatId),
		SenderId:    senderChatId,
		SenderName:  GetSenderName(upd.Message.Sender),
		Content:     content,
		Attachments: GetContentStructs(upd.Message.Content),
		ContentRaw:  nil,
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

func parseUpdateDeleteMessages(upd *client.UpdateDeleteMessages, date int32) structs.DeleteMessages {
	result := structs.DeleteMessages{
		T:          "DeleteMessages",
		MessageIds: upd.MessageIds,
		ChatId:     upd.ChatId,
		ChatName:   GetChatName(upd.ChatId),
		Date:       date,
		DateStr:    FormatTime(date),
	}
	for _, messageId := range upd.MessageIds {
		m, err := FindUpdateNewMessage(messageId)
		if err != nil {
			result.Messages = append(result.Messages, structs.MessageError{T: "Error", MessageId: messageId, Error: fmt.Sprintf("not found deleted message %s", err)})
			continue
		}
		result.Messages = append(result.Messages, parseUpdateNewMessage(m))
	}

	return result
}

func formatMessageLink(chatId int64, messageId int64) string {
	link := GetLink(chatId, messageId)
	if link != "" {
		return fmt.Sprintf(`<a href="%s">message</a>`, link)
	} else {
		return "message"
	}
}

func formatNewMessageLink(upd *client.UpdateNewMessage) string {
	chat, _ := GetChat(upd.Message.ChatId)
	link := formatMessageLink(upd.Message.ChatId, upd.Message.Id)
	if upd.Message.Sender.MessageSenderType() == "messageSenderChat" {
		return fmt.Sprintf(`<a href="/e/%d/%d">new</a> %s in channel <a href="/c/%d">%s</a>`, upd.Message.ChatId, upd.Message.Id, link, chat.Id, chat.Title)
	} else {
		return fmt.Sprintf(`<a href="/e/%d/%d">new</a> %s from <a href="/c/%d">%s</a> in chat <a href="/c/%d">%s</a>`, upd.Message.ChatId, upd.Message.Id, link, GetChatIdBySender(upd.Message.Sender), GetSenderName(upd.Message.Sender), chat.Id, chat.Title)
	}
}

func formatDeletedMessagesLink(upd *client.UpdateDeleteMessages) string {
	chat, _ := GetChat(upd.ChatId)

	return fmt.Sprintf(`<a href="/d/%d/%s">deleted</a> %d messages from chat <a href="/c/%d">%s</a>`, upd.ChatId, ImplodeInt(upd.MessageIds), len(upd.MessageIds), chat.Id, chat.Title)
}

func formatUpdatedContentLink(upd *client.UpdateMessageContent) string {
	chat, _ := GetChat(upd.ChatId)
	m, err := FindUpdateNewMessage(upd.MessageId)
	link := formatMessageLink(upd.ChatId, upd.MessageId)
	if err != nil {

		return fmt.Sprintf(`<a href="/e/%d/%d">updated</a> %s in chat <a href="/c/%d">%s</a>`, upd.ChatId, upd.MessageId, link, chat.Id, chat.Title)
	}

	if m.Message.Sender.MessageSenderType() == "messageSenderChat" {
		return fmt.Sprintf(`<a href="/e/%d/%d">updated</a> %s in channel <a href="/c/%d">%s</a>`, m.Message.ChatId, m.Message.Id, link, chat.Id, chat.Title)
	} else {
		return fmt.Sprintf(`<a href="/e/%d/%d">updated</a> %s from <a href="/c/%d">%s</a> in chat <a href="/c/%d">%s</a>`, m.Message.ChatId, m.Message.Id, link, GetChatIdBySender(m.Message.Sender), GetSenderName(m.Message.Sender), chat.Id, chat.Title)
	}
}
