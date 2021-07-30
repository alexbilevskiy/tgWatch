package libs

import (
	"fmt"
	"go-tdlib/client"
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
				IntLink: fmt.Sprintf("/m/%d/%d", upd.Message.ChatId, upd.Message.Id), //@TODO: link shoud be /m
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
				IntLink: fmt.Sprintf("/m/%d/%d", upd.ChatId, upd.MessageId),
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
				IntLink: fmt.Sprintf("/m/%d/%d", upd.ChatId, upd.MessageId),
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
				IntLink: fmt.Sprintf("/h/%d/?ids=%s", upd.ChatId, ImplodeInt(upd.MessageIds)),
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

func processTgActiveSessions(w http.ResponseWriter) {
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

func processTgSingleMessage(chatId int64, messageId int64, w http.ResponseWriter) {
	upd, err := FindUpdateNewMessage(chatId, messageId)
	if err != nil {
		m := structs.MessageError{T: "Error", MessageId: messageId, Error: fmt.Sprintf("Error: %s", err)}
		renderTemplates(w, m, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/error.tmpl`)

		return
	}

	senderChatId := GetChatIdBySender(upd.Message.Sender)
	ct := GetContentWithText(upd.Message.Content, upd.Message.ChatId)
	msg := structs.MessageInfo{
		T:             "NewMessage",
		MessageId:     upd.Message.Id,
		Date:          upd.Message.Date,
		ChatId:        upd.Message.ChatId,
		ChatName:      GetChatName(upd.Message.ChatId),
		SenderId:      senderChatId,
		SenderName:    GetSenderName(upd.Message.Sender),
		MediaAlbumId:  int64(upd.Message.MediaAlbumId),
		SimpleText:    ct.Text,
		FormattedText: ct.FormattedText,
		Attachments:   GetContentAttachments(upd.Message.Content),
		Deleted:       IsMessageDeleted(upd.Message.ChatId, upd.Message.Id),
		Edited:        IsMessageEdited(upd.Message.ChatId, upd.Message.Id),
		ContentRaw:    nil,
	}
	chat, _ := GetChat(upd.Message.ChatId, false)
	res := structs.SingleMessage{
		T: "Message",
		Message: msg,
		Edits: make([]structs.MessageEditedInfo, 0),
		Chat: buildChatInfoByLocalChat(chat, false),
	}

	updates, updateTypes, dates, err := FindAllMessageChanges(chatId, messageId)
	if err != nil {
		m := structs.MessageError{T: "Error", MessageId: messageId, Error: fmt.Sprintf("Error: %s", err)}
		renderTemplates(w, m, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/error.tmpl`)
		return
	}

	var edit structs.MessageEditedInfo
	for i, rawJsonBytes := range updates {
		switch updateTypes[i] {
		case client.TypeUpdateNewMessage:
		case client.TypeUpdateMessageEdited:
			upd, _ := client.UnmarshalUpdateMessageEdited(rawJsonBytes)
			edit.MessageId = upd.MessageId
			edit.Date = upd.EditDate
			res.Edits = append(res.Edits, edit)

			break
		case client.TypeUpdateMessageContent:
			upd, _ := client.UnmarshalUpdateMessageContent(rawJsonBytes)
			ct = GetContentWithText(upd.NewContent, upd.ChatId)
			edit = structs.MessageEditedInfo{T:"MessageEdited"}
			edit.FormattedText = ct.FormattedText
			edit.SimpleText = ct.Text
			edit.Attachments = GetContentAttachments(upd.NewContent)

			break
		case client.TypeUpdateDeleteMessages:
			//upd, _ := client.UnmarshalUpdateDeleteMessages(rawJsonBytes)
			msg.DeletedAt = dates[i]
			break
		default:
			//m := structs.MessageError{T:"Error", MessageId: messageId, Error: fmt.Sprintf("Unknown update type: %s", updateTypes[i])}
			//fullContentJ = append(fullContentJ, m)
		}
	}



	renderTemplates(w, res, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/single_message.tmpl`, `templates/message.tmpl`)
}

func processTgMessagesByIds(chatId int64, messageIds []int64, w http.ResponseWriter) {
	res := structs.ChatHistory{
		T: "ChatHistory-filtered",
		Messages: make([]structs.MessageInfo, 0),
	}

	for _, messageId := range messageIds {
		upd, err := FindUpdateNewMessage(chatId, messageId)
		if err != nil {
			m := structs.MessageInfo{T: "Error", MessageId: messageId, SimpleText: fmt.Sprintf("Error: %s", err)}
			res.Messages = append(res.Messages, m)

			continue
		}

		res.Messages = append(res.Messages, parseUpdateNewMessage(upd))
	}

	renderTemplates(w, res, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/chat_history_filtered.tmpl`, `templates/messages_list.tmpl`, `templates/message.tmpl`)
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

func processTgChatHistory(chatId int64, limit int64, offset int64, deleted bool, w http.ResponseWriter) {
	updates, _, _, errSelect := GetChatHistory(chatId, limit, offset, deleted)
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

	for _, rawJsonBytes := range updates {
		upd, _ := client.UnmarshalUpdateNewMessage(rawJsonBytes)
		msg := parseUpdateNewMessage(upd)
		//hack to reverse, orig was: res.Messages = append(res.Messages, msg)
		res.Messages = append([]structs.MessageInfo{msg}, res.Messages...)
	}

	renderTemplates(w, res, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/chat_history.tmpl`, `templates/messages_list.tmpl`, `templates/message.tmpl`)
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
			content := GetContentWithText(message.Content, message.ChatId).Text
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

func processSettings(r *http.Request, w http.ResponseWriter) {
	var res structs.IgnoreLists
	if r.Method == "POST" {
		//@TODO: VALIDATE FORM DATA!! Only int acceptable as chat ID, only valid names for folders
		IgnoreChatIds := make(map[string]bool, 0)
		if _, ok := r.PostForm["ignoreChatIds"]; ok {
			for _, chatId := range r.PostForm["ignoreChatIds"] {
				if chatId == "" {
					continue
				}
				IgnoreChatIds[chatId] = true
			}
		}
		IgnoreAuthorIds := make(map[string]bool, 0)
		if _, ok := r.PostForm["ignoreAuthorIds"]; ok {
			for _, authorId := range r.PostForm["ignoreAuthorIds"] {
				if authorId == "" {
					continue
				}
				IgnoreAuthorIds[authorId] = true
			}
		}
		IgnoreFolders := make(map[string]bool, 0)
		if _, ok := r.PostForm["ignoreFolders"]; ok {
			for _, folder := range r.PostForm["ignoreFolders"] {
				if folder == "" {
					continue
				}
				IgnoreFolders[folder] = true
			}
		}
		ignoreLists.IgnoreChatIds = IgnoreChatIds
		ignoreLists.IgnoreAuthorIds = IgnoreAuthorIds
		ignoreLists.IgnoreFolders = IgnoreFolders
		saveSettings()
		res = ignoreLists
		res.T = "Settings"

	} else {
		res = ignoreLists
		res.T = "Settings"
	}

	renderTemplates(w, res, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/settings.tmpl`)
}