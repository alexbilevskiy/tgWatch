package libs

import (
	"encoding/json"
	"fmt"
	"go-tdlib/client"
	"html/template"
	"net/http"
	"strings"
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


func processTgJournal(limit int64, w http.ResponseWriter)  {
	updates, updateTypes, dates, errSelect := FindRecentChanges(limit)
	if errSelect != nil {
		fmt.Printf("Error select updates: %s\n", errSelect)

		return
	}
	var t *template.Template
	var errParse error
	if verbose {
		t, errParse = template.New(`json.tmpl`).ParseFiles(`templates/json.tmpl`)
	} else {
		t, errParse = template.New(`base.tmpl`).ParseFiles(`templates/base.tmpl`, `templates/navbar.tmpl`, `templates/journal.tmpl`)
	}

	if errParse != nil {
		fmt.Printf("Error parse tpl: %s\n", errParse)
		return
	}
	var data structs.Journal
	data.T = "Journal"

	for i, rawJsonBytes := range updates {
		switch updateTypes[i] {
		case "updateNewMessage":
			upd, _ := client.UnmarshalUpdateNewMessage(rawJsonBytes)
			item := structs.JournalItem{
				T: updateTypes[i],
				Time: dates[i],
				Date: FormatTime(dates[i]),
				Link: GetLink(upd.Message.ChatId, upd.Message.Id),
				IntLink: fmt.Sprintf("/e/%d/%d", upd.Message.ChatId, upd.Message.Id), //@TODO: link shoud be /m
				Chat: structs.ChatInfo{
					ChatId: upd.Message.ChatId,
					ChatName: GetChatName(upd.Message.ChatId),
				},
			}
			if upd.Message.Sender.MessageSenderType() == "messageSenderChat" {
			} else {
				item.From = structs.ChatInfo{ChatId: GetChatIdBySender(upd.Message.Sender), ChatName: GetSenderName(upd.Message.Sender)}
			}
			data.J = append(data.J, item)

			break
		case "updateMessageEdited":
			upd, _ := client.UnmarshalUpdateMessageEdited(rawJsonBytes)
			item := structs.JournalItem{
				T: updateTypes[i],
				Time: dates[i],
				Date: FormatTime(dates[i]),
				Link: GetLink(upd.ChatId, upd.MessageId),
				IntLink: fmt.Sprintf("/e/%d/%d", upd.ChatId, upd.MessageId),
				Chat: structs.ChatInfo{
					ChatId: upd.ChatId,
					ChatName: GetChatName(upd.ChatId),
				},
			}
			data.J = append(data.J, item)

			break
		case "updateMessageContent":
			upd, _ := client.UnmarshalUpdateMessageContent(rawJsonBytes)
			item := structs.JournalItem{
				T: updateTypes[i],
				Time: dates[i],
				Date: FormatTime(dates[i]),
				Link: GetLink(upd.ChatId, upd.MessageId),
				IntLink: fmt.Sprintf("/e/%d/%d", upd.ChatId, upd.MessageId),
				Chat: structs.ChatInfo{
					ChatId: upd.ChatId,
					ChatName: GetChatName(upd.ChatId),
				},
			}
			m, err := FindUpdateNewMessage(upd.ChatId, upd.MessageId)
			if err != nil {
				item.Error = fmt.Sprintf("Message not found: %s", err)
				data.J = append(data.J, item)

				break
			}

			if m.Message.Sender.MessageSenderType() == "messageSenderChat" {
			} else {
				item.From = structs.ChatInfo{ChatId: GetChatIdBySender(m.Message.Sender), ChatName: GetSenderName(m.Message.Sender)}
			}
			data.J = append(data.J, item)

			break
		case "updateDeleteMessages":
			upd, _ := client.UnmarshalUpdateDeleteMessages(rawJsonBytes)
			item := structs.JournalItem{
				T: updateTypes[i],
				Time: dates[i],
				Date: FormatTime(dates[i]),
				IntLink: fmt.Sprintf("/d/%d/%s", upd.ChatId, ImplodeInt(upd.MessageIds)),
				Chat: structs.ChatInfo{
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
		err = t.Execute(w, structs.JSON{JSON: JsonMarshalStr(data)})
	} else {
		err = t.Execute(w, data)
	}

	if err != nil {
		fmt.Printf("Error tpl: %s\n", err)
		return
	}
}

func processTgOverview(limit int64, w http.ResponseWriter) {
	s, err := GetChatsStats()
	if err != nil {
		fmt.Printf("Error tpl: %s\n", err)

		return
	}
	var t *template.Template
	var errParse error
	if verbose {
		t, errParse = template.New(`json.tmpl`).ParseFiles(`templates/json.tmpl`)
	} else {
		t, errParse = template.New(`base.tmpl`).ParseFiles(`templates/base.tmpl`, `templates/navbar.tmpl`, `templates/overview.tmpl`)
	}
	if errParse != nil {
		fmt.Printf("Error tpl: %s\n", errParse)

		return
	}

	data := structs.Overview{T: "Overview"}
	for _, ci := range s {
		oi := structs.OverviewItem{
			Chat: structs.ChatInfo{
				ChatId: ci.ChatId,
				ChatName: GetChatName(ci.ChatId),
			},
			CountTotal: ci.Counters["total"],
			CountMessages: ci.Counters["updateNewMessage"],
			CountDeletes: ci.Counters["updateDeleteMessages"],
			CountEdits: ci.Counters["updateMessageEdited"],
		}
		data.O = append(data.O, oi)
	}
	if verbose {
		err = t.Execute(w, structs.JSON{JSON: JsonMarshalStr(data)})
	} else {
		err = t.Execute(w, data)
	}

	if err != nil {
		fmt.Printf("Error tpl: %s\n", err)
		return
	}
}

func processSingleMessage(chatId int64, messageId int64, w http.ResponseWriter) {
	verbose = !verbose

	var t *template.Template
	var errParse error
	if verbose {
		t, errParse = template.New(`json.tmpl`).ParseFiles(`templates/json.tmpl`)
	} else {
		t, errParse = template.New(`base.tmpl`).ParseFiles(`templates/base.tmpl`, `templates/navbar.tmpl`, `templates/single_message.tmpl`)
	}
	if errParse != nil {
		fmt.Printf("Error tpl: %s\n", errParse)

		return
	}

	message, err := FindUpdateNewMessage(chatId, messageId)
	if err != nil {
		fmt.Printf("Not found message %s", err)

		return
	}
	if verbose {
		err = t.Execute(w, structs.JSON{JSON: JsonMarshalStr(message)})
	} else {
		err = t.Execute(w, parseUpdateNewMessage(message))
	}

	if err != nil {
		fmt.Printf("Error tpl: %s\n", err)

		return
	}
}

func processTgDeleted(chatId int64, messageIds []int64) []byte {

	var fullContentJ []interface{}
	for _, messageId := range messageIds {
		upd, err := FindUpdateNewMessage(chatId, messageId)
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

	updates, updateTypes, dates, err := FindAllMessageChanges(chatId, messageId)
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

func processTgChatInfo(chatId int64, w http.ResponseWriter) {
	var err error
	var t *template.Template

	if verbose {
		t, err = template.New(`json.tmpl`).ParseFiles(`templates/json.tmpl`)
	} else {
		t, err = template.New(`base.tmpl`).ParseFiles(`templates/base.tmpl`, `templates/navbar.tmpl`, `templates/chat_info.tmpl`)
	}

	if err != nil {
		fmt.Printf("Error parse tpl: %s\n", err)
		return
	}

	var chat interface{}
	if chatId > 0 {
		chat, err = GetUser(int32(chatId))
	} else{
		chat, err = GetChat(chatId, false)
	}
	if err != nil {
		fmt.Printf("Error get chat: %s\n", err)
		return
	}

	res := structs.ChatFullInfo{
		T: "ChatFullInfo",
		Chat: chat,
		ChatRaw: jsonMarshalPretty(chat),
	}
	if verbose {
		err = t.Execute(w, structs.JSON{JSON: JsonMarshalStr(res)})

		return
	} else {
		err = t.Execute(w, res)
	}
	if err != nil {
		fmt.Printf("Error tpl: %s\n", err)
		return
	}

}

func processTgChatHistory(chatId int64, limit int64, w http.ResponseWriter) {
	updates, updateTypes, _, errSelect := GetChatHistory(chatId, limit)
	if errSelect != nil {
		fmt.Printf("Error select updates: %s\n", errSelect)

		return
	}
	var t *template.Template
	var err error
	if verbose {
		t, err = template.New(`json.tmpl`).ParseFiles(`templates/json.tmpl`)
	} else {
		t, err = template.New(`base.tmpl`).ParseFiles(`templates/base.tmpl`, `templates/navbar.tmpl`, `templates/chat_history.tmpl`)
	}

	if err != nil {
		fmt.Printf("Error parse tpl: %s\n", err)
		return
	}

	res := structs.ChatHistory{
		T: "ChatHistory",
		Chat: structs.ChatInfo{
			ChatId:   chatId,
			ChatName: GetChatName(chatId),
		},
	}

	for i, rawJsonBytes := range updates {
		switch updateTypes[i] {
		case "updateNewMessage":
			upd, _ := client.UnmarshalUpdateNewMessage(rawJsonBytes)
			senderChatId := GetChatIdBySender(upd.Message.Sender)
			content := GetContent(upd.Message.Content)
			msg := structs.MessageInfo{
				T:            "NewMessage",
				MessageId:    upd.Message.Id,
				Date:         upd.Message.Date,
				DateStr:      FormatTime(upd.Message.Date),
				ChatId:       upd.Message.ChatId,
				ChatName:     GetChatName(upd.Message.ChatId),
				SenderId:     senderChatId,
				SenderName:   GetSenderName(upd.Message.Sender),
				MediaAlbumId: int64(upd.Message.MediaAlbumId),
				Content:      content,
				Attachments:  GetContentStructs(upd.Message.Content),
				ContentRaw:   nil,
			}
			res.Messages = append(res.Messages, msg)

			break
		default:
			fmt.Printf("Not supported chat history item %s\n", updateTypes[i])
		}
	}
	if verbose {
		err = t.Execute(w, structs.JSON{JSON: JsonMarshalStr(res)})

		return
	} else {
		err = t.Execute(w, res)
	}
	if err != nil {
		fmt.Printf("Error tpl: %s\n", err)
		return
	}

	return
}

func processTgChatList(w http.ResponseWriter, refresh bool, folder int32) {
	var t *template.Template
	var err error
	if verbose {
		t, err = template.New(`json.tmpl`).ParseFiles(`templates/json.tmpl`)
	} else {
		t, err = template.New(`base.tmpl`).ParseFiles(`templates/base.tmpl`, `templates/navbar.tmpl`, `templates/chatlist.tmpl`)
	}

	if err != nil {
		fmt.Printf("Error parse tpl: %s\n", err)
		return
	}
	var folders []structs.ChatFolder
	folders = make([]structs.ChatFolder, 0)
	folders = append(folders, structs.ChatFolder{T: "ChatFolder", Id: ClMain, Title: "Main"})
	folders = append(folders, structs.ChatFolder{T: "ChatFolder", Id: ClArchive, Title: "Archive"})
	for _, filter := range chatFilters {
		folders = append(folders, structs.ChatFolder{T: "ChatFolder", Id: filter.Id, Title: filter.Title})
	}

	res := structs.ChatList{T: "Chat list", ChatFolders: folders, SelectedFolder: folder}
	if refresh {
		chatList := getChatsList(folder)
		for _, chat := range chatList {
			res.Chats = append(res.Chats, structs.ChatInfo{ChatId: chat.Id, ChatName: GetChatName(chat.Id)})
		}
	} else {
		chatList := getSavedChats(folder)
		for _, chatPos := range chatList {
			res.Chats = append(res.Chats, structs.ChatInfo{ChatId: chatPos.ChatId, ChatName: GetChatName(chatPos.ChatId)})
		}
	}

	if verbose {
		err = t.Execute(w, structs.JSON{JSON: JsonMarshalStr(res)})

		return
	} else {
		err = t.Execute(w, res)
	}
	if err != nil {
		fmt.Printf("Error tpl: %s\n", err)
		return
	}

	return
}

func processTgDelete(chatId int64, pattern string, w http.ResponseWriter) {

	req := &client.GetChatHistoryRequest{ChatId: chatId, Limit: 100, FromMessageId: 0, Offset: 0}
	history, err := tdlibClient.GetChatHistory(req)
	if err != nil {
		data := []byte(fmt.Sprintf("Error get chat %d history: %s", chatId, err))
		w.Write(data)
		return
	}
	var messageIds []int64
	messageIds = make([]int64, 0)
	fmt.Printf("Received history of %d messages from chat %d\n", history.TotalCount, chatId)
	for _, message := range history.Messages {
		content := GetContent(message.Content)
		if content == "" {
			fmt.Printf("NO content: %d, `%s`\n", message.Id, content)
			continue
		}
		if strings.Contains(content, pattern) {
			fmt.Printf("Delete candidate: %d, `%s`\n", message.Id, content)
			messageIds = append(messageIds, message.Id)
		} else {
			fmt.Printf("SKIP: %d, `%s`\n", message.Id, content)
		}
	}
	reqDelete := &client.DeleteMessagesRequest{ChatId: chatId, MessageIds: messageIds}
	ok, err := tdlibClient.DeleteMessages(reqDelete)
	if err != nil {
		fmt.Printf("Failed to delete: `%s`\n", err)
		return
	}
	if ok != nil {
		fmt.Printf("Deleted batch of: %d messages\n", len(messageIds))
		data := []byte(fmt.Sprintf("Deleted from chat %d `%s`", chatId, pattern))
		w.Write(data)
		return
	}
	data := []byte(fmt.Sprintf("Deleted from chat %d `%s`", chatId, pattern))
	w.Write(data)
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
		T:            "NewMessage",
		MessageId:    upd.Message.Id,
		Date:         upd.Message.Date,
		DateStr:      FormatTime(upd.Message.Date),
		ChatId:       upd.Message.ChatId,
		ChatName:     GetChatName(upd.Message.ChatId),
		SenderId:     senderChatId,
		SenderName:   GetSenderName(upd.Message.Sender),
		Content:      content,
		Attachments:  GetContentStructs(upd.Message.Content),
		ContentRaw:   nil,
		MediaAlbumId: int64(upd.Message.MediaAlbumId),
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
		m, err := FindUpdateNewMessage(upd.ChatId, messageId)
		if err != nil {
			result.Messages = append(result.Messages, structs.MessageError{T: "Error", MessageId: messageId, Error: fmt.Sprintf("not found deleted message %s", err)})
			continue
		}
		result.Messages = append(result.Messages, parseUpdateNewMessage(m))
	}

	return result
}
