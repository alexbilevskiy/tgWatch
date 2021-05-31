package libs

import (
	"fmt"
	"go-tdlib/client"
	"html/template"
	"net/http"
	"strings"
	"tgWatch/structs"
)

func processTgJournal(limit int64, w http.ResponseWriter)  {
	updates, updateTypes, dates, errSelect := FindRecentChanges(limit)
	if errSelect != nil {
		fmt.Printf("Error select updates: %s\n", errSelect)

		return
	}
	var data structs.Journal
	data.T = "Journal"

	for i, rawJsonBytes := range updates {
		switch updateTypes[i] {
		case "updateNewMessage":
			upd, _ := client.UnmarshalUpdateNewMessage(rawJsonBytes)
			item := structs.JournalItem{
				T:       updateTypes[i],
				Time:    dates[i],
				Date:    FormatDateTime(dates[i]),
				Link:    GetLink(upd.Message.ChatId, upd.Message.Id),
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
				T:       updateTypes[i],
				Time:    dates[i],
				Date:    FormatDateTime(dates[i]),
				Link:    GetLink(upd.ChatId, upd.MessageId),
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
				T:       updateTypes[i],
				Time:    dates[i],
				Date:    FormatDateTime(dates[i]),
				Link:    GetLink(upd.ChatId, upd.MessageId),
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
				T:       updateTypes[i],
				Time:    dates[i],
				Date:    FormatDateTime(dates[i]),
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
			//fc += fmt.Sprintf("[%s] Unknown update type \"%s\"<br>", FormatDateTime(dates[i]), updateTypes[i])
		}
	}
	renderTemplates(w, data, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/journal.tmpl`)
}

func processTgOverview(limit int64, w http.ResponseWriter) {
	s, err := GetChatsStats(make([]int64, 0))
	if err != nil {
		fmt.Printf("Error tpl: %s\n", err)

		return
	}
	data := structs.Overview{T: "Overview"}
	for _, ci := range s {
		oi := structs.ChatInfo{
			ChatId: ci.ChatId,
			ChatName: GetChatName(ci.ChatId),
			CountTotal: ci.Counters["total"],
			CountMessages: ci.Counters["updateNewMessage"],
			CountDeletes: ci.Counters["updateDeleteMessages"],
			CountEdits: ci.Counters["updateMessageEdited"],
		}
		data.Chats = append(data.Chats, oi)
	}
	renderTemplates(w, data, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/overview_table.tmpl`, `templates/overview.tmpl`)
}

func processTdlibOptions(w http.ResponseWriter) {
	actualOptions := make(map[string]structs.TdlibOption, len(tdlibOptions))
	for optionName, optionValue := range tdlibOptions {
		req := client.GetOptionRequest{Name: optionName}
		res, err := tdlibClient.GetOption(&req)
		if err != nil {
			fmt.Printf("Failed to get option %s: %s", optionName, err)
			continue
		}

		switch res.OptionValueType() {
		case client.TypeOptionValueInteger:
			actualOption := res.(*client.OptionValueInteger)
			optionValue.Value = int64(actualOption.Value)
		case client.TypeOptionValueString:
			actualOption := res.(*client.OptionValueString)
			optionValue.Value = string(actualOption.Value)
		case client.TypeOptionValueBoolean:
			actualOption := res.(*client.OptionValueBoolean)
			optionValue.Value = bool(actualOption.Value)
		case client.TypeOptionValueEmpty:
			optionValue.Value = nil
		}
		actualOptions[optionName] = optionValue
	}
	data := structs.OptionsList{T: "OptionsLists", Options: actualOptions}
	renderTemplates(w, data, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/tdlib_options.tmpl`)
}

func processActiveSessions(w http.ResponseWriter) {
	sessions, err := tdlibClient.GetActiveSessions()
	if err != nil {
		fmt.Printf("Get sessions error: %s", err)
		return
	}
	data := structs.SessionsList{T:"Sessions", Sessions: sessions}
	if !verbose {
		data.SessionsRaw = jsonMarshalPretty(sessions)
	}

	renderTemplates(w, data, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/sessions_list.tmpl`)
}

func processSingleMessage(chatId int64, messageId int64, w http.ResponseWriter) {
	verbose = !verbose
	message, err := FindUpdateNewMessage(chatId, messageId)
	if err != nil {
		fmt.Printf("Not found message %s", err)

		return
	}

	var data interface{}
	if verbose {
		data = message
	} else {
		data = parseUpdateNewMessage(message)
	}

	renderTemplates(w, data, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/single_message.tmpl`)
}

func processTgDeleted(chatId int64, messageIds []int64, w http.ResponseWriter) {
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

	res := structs.Messages{
		T: "DeletedMessages",
		Messages: fullContentJ,
	}
	if !verbose {
		res.MessagesRaw = jsonMarshalPretty(fullContentJ)
	}

	renderTemplates(w, res, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/deleted_message.tmpl`)
}

func processTgEdit(chatId int64, messageId int64, w http.ResponseWriter) {
	var fullContentJ []interface{}

	updates, updateTypes, dates, err := FindAllMessageChanges(chatId, messageId)
	if err != nil {
		m := structs.MessageError{T: "Error", MessageId: messageId, Error: fmt.Sprintf("Error: %s", err)}
		renderTemplates(w, m, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/error.tmpl`)
		return
	}

	for i, rawJsonBytes := range updates {
		switch updateTypes[i] {
		case client.TypeUpdateNewMessage:
			upd, _ := client.UnmarshalUpdateNewMessage(rawJsonBytes)
			fullContentJ = append(fullContentJ, parseUpdateNewMessage(upd))
			break
		case client.TypeUpdateMessageEdited:
			upd, _ := client.UnmarshalUpdateMessageEdited(rawJsonBytes)
			fullContentJ = append(fullContentJ, parseUpdateMessageEdited(upd))
			break
		case client.TypeUpdateMessageContent:
			upd, _ := client.UnmarshalUpdateMessageContent(rawJsonBytes)
			fullContentJ = append(fullContentJ, parseUpdateMessageContent(upd))
			break
		case client.TypeUpdateDeleteMessages:
			upd, _ := client.UnmarshalUpdateDeleteMessages(rawJsonBytes)
			fullContentJ = append(fullContentJ, parseUpdateDeleteMessages(upd, dates[i]))
			break
		default:
			m := structs.MessageError{T:"Error", MessageId: messageId, Error: fmt.Sprintf("Unknown update type: %s", updateTypes[i])}
			fullContentJ = append(fullContentJ, m)
		}
	}

	res := structs.Messages{
		T: "EditedMessages",
		Messages: fullContentJ,
	}
	if !verbose {
		res.MessagesRaw = jsonMarshalPretty(fullContentJ)
	}

	renderTemplates(w, res, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/edited_message.tmpl`)
}

func processTgChatInfo(chatId int64, w http.ResponseWriter) {
	var chat interface{}
	var err error
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
	}
	var data interface{}
	if verbose {
		data = res
	} else {
		res.ChatRaw = jsonMarshalPretty(chat)
		data = res
	}
	renderTemplates(w, data, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/chat_info.tmpl`)
}

func processTgChatHistory(chatId int64, limit int64, offset int64, w http.ResponseWriter) {
	updates, updateTypes, _, errSelect := GetChatHistory(chatId, limit, offset)
	if errSelect != nil {
		fmt.Printf("Error select updates: %s\n", errSelect)

		return
	}
	chat, _ := GetChat(chatId, false)
	res := structs.ChatHistory{
		T: "ChatHistory",
		Chat: buildChatInfoByLocalChat(chat, false),
		Limit:  limit,
		Offset: offset,
		NextOffset: offset + limit,
		PrevOffset: offset - limit,
	}
	if res.PrevOffset < 0 {
		res.PrevOffset = 0
	}

	for i, rawJsonBytes := range updates {
		switch updateTypes[i] {
		case client.TypeUpdateNewMessage:
			upd, _ := client.UnmarshalUpdateNewMessage(rawJsonBytes)
			senderChatId := GetChatIdBySender(upd.Message.Sender)
			content := GetContent(upd.Message.Content)
			msg := structs.MessageInfo{
				T:            "NewMessage",
				MessageId:    upd.Message.Id,
				Date:         upd.Message.Date,
				DateTimeStr:  FormatDateTime(upd.Message.Date),
				DateStr:      FormatDate(upd.Message.Date),
				TimeStr:      FormatTime(upd.Message.Date),
				ChatId:       upd.Message.ChatId,
				ChatName:     GetChatName(upd.Message.ChatId),
				SenderId:     senderChatId,
				SenderName:   GetSenderName(upd.Message.Sender),
				MediaAlbumId: int64(upd.Message.MediaAlbumId),
				Content:      content,
				Attachments:  GetContentStructs(upd.Message.Content),
				ContentRaw:   nil,
			}
			//hack to reverse, orig was: res.Messages = append(res.Messages, msg)
			res.Messages = append([]structs.MessageInfo{msg}, res.Messages...)

			break
		default:
			fmt.Printf("Not supported chat history item %s\n", updateTypes[i])
		}
	}

	renderTemplates(w, res, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/chat_history.tmpl`)
}

func processTgChatList(refresh bool, folder int32, w http.ResponseWriter) {
	var folders []structs.ChatFolder
	folders = make([]structs.ChatFolder, 0)
	folders = append(folders, structs.ChatFolder{T: "ChatFolder", Id: ClMain, Title: "Main"})
	folders = append(folders, structs.ChatFolder{T: "ChatFolder", Id: ClArchive, Title: "Archive"})
	folders = append(folders, structs.ChatFolder{T: "ChatFolder", Id: ClCached, Title: "Cached"})
	folders = append(folders, structs.ChatFolder{T: "ChatFolder", Id: ClMy, Title: "Owned chats"})
	folders = append(folders, structs.ChatFolder{T: "ChatFolder", Id: ClNotSubscribed, Title: "Not subscribed chats"})
	for _, filter := range chatFilters {
		folders = append(folders, structs.ChatFolder{T: "ChatFolder", Id: filter.Id, Title: filter.Title})
	}

	res := structs.ChatList{T: "Chat list", ChatFolders: folders, SelectedFolder: folder}
	if folder == ClCached {
		for _, chat := range localChats {
			info := buildChatInfoByLocalChat(chat, true)
			res.Chats = append(res.Chats, info)
		}
	} else if folder == ClMy {
		for _, chat := range localChats {
			req := &client.GetChatMemberRequest{ChatId: chat.Id, UserId: me.Id}
			cm, err := tdlibClient.GetChatMember(req)
			if err != nil {
				fmt.Printf("failed to get chat member status: %d, `%s`, %s\n", chat.Id, GetChatName(chat.Id), err)
				continue
			}
			switch cm.Status.ChatMemberStatusType() {
			case client.TypeChatMemberStatusCreator:
				res.Chats = append(res.Chats, structs.ChatInfo{ChatId: chat.Id, ChatName: GetChatName(chat.Id)})
			case client.TypeChatMemberStatusAdministrator:
			case client.TypeChatMemberStatusMember:
			case client.TypeChatMemberStatusLeft:
			default:
				fmt.Printf("Unusual chat memer status: %d, `%s`, %s\n", chat.Id, GetChatName(chat.Id), cm.Status.ChatMemberStatusType())

			}
		}
	} else if folder == ClNotSubscribed {
		for _, chat := range localChats {
			if chat.LastMessage == nil && chat.LastReadInboxMessageId == 0 {
				info := buildChatInfoByLocalChat(chat, true)
				res.Chats = append(res.Chats, info)
			}
		}
	} else if refresh {
		chatList := getChatsList(folder)
		for _, chat := range chatList {
			res.Chats = append(res.Chats, buildChatInfoByLocalChat(chat, false))
		}
	} else {
		chatList := getSavedChats(folder)
		for _, chatPos := range chatList {
			chat, err := GetChat(chatPos.ChatId, false)
			var chatInfo structs.ChatInfo
			if err != nil {
				chatInfo = structs.ChatInfo{ChatId: chatPos.ChatId, ChatName: GetChatName(chatPos.ChatId), Username: "ERROR " + err.Error()}
			} else {
				chatInfo = buildChatInfoByLocalChat(chat, true)
			}

			res.Chats = append(res.Chats, chatInfo)
		}
	}
	renderTemplates(w, res, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/overview_table.tmpl`, `templates/chatlist.tmpl`)
}

func processTgDelete(chatId int64, pattern string, limit int, w http.ResponseWriter) {

	var messageIds []int64
	messageIds = make([]int64, 0)
	var lastId int64 = 0
	for len(messageIds) < limit {
		req := &client.GetChatHistoryRequest{ChatId: chatId, Limit: 100, FromMessageId: lastId, Offset: 0}
		history, err := tdlibClient.GetChatHistory(req)
		if err != nil {
			data := []byte(fmt.Sprintf("Error get chat %d history: %s", chatId, err))
			w.Write(data)
			return
		}
		fmt.Printf("Received history of %d messages from chat %d\n", history.TotalCount, chatId)
		noMore := true
		for _, message := range history.Messages {
			lastId = message.Id
			content := GetContent(message.Content)
			if content == "" {
				fmt.Printf("NO content: %d, `%s`\n", message.Id, content)
				continue
			}
			if strings.Contains(content, pattern) {
				fmt.Printf("Delete candidate: %d\n", message.Id)
				messageIds = append(messageIds, message.Id)
				noMore = false
			} else {
				fmt.Printf("SKIP: %d, `%s`\n", message.Id, content)
			}
			if noMore {
				break
			}
		}
	}
	reqDelete := &client.DeleteMessagesRequest{ChatId: chatId, MessageIds: messageIds}
	var ok *client.Ok
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

func renderTemplates(w http.ResponseWriter, templateData interface{}, templates... string) {
	var t *template.Template
	var errParse error
	if verbose {
		t, errParse = template.New(`json.tmpl`).ParseFiles(`templates/json.tmpl`)
	} else {
		t, errParse = template.New(`base.tmpl`).Funcs(template.FuncMap{
			"safeHTML": func(b string) template.HTML {
				return template.HTML(b)
			},
			"isMe": func(chatId int64) bool {
				if chatId == int64(me.Id) {

					return true
				}

				return false
			},
		},
		).ParseFiles(templates...)
	}
	if errParse != nil {
		fmt.Printf("Error tpl: %s\n", errParse)

		return
	}

	var err error
	if verbose {
		err = t.Execute(w, structs.JSON{JSON: JsonMarshalStr(templateData)})
	} else {
		err = t.Execute(w, templateData)
	}

	if err != nil {
		fmt.Printf("Error tpl: %s\n", err)

		return
	}

}