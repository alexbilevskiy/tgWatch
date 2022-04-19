package libs

import (
	"fmt"
	"go-tdlib/client"
	"net/http"
	"os"
	"strconv"
	"strings"
	"tgWatch/structs"
)

func processTgJournal(req *http.Request, w http.ResponseWriter) {
	limit := int64(50)
	if req.FormValue("limit") != "" {
		limit, _ = strconv.ParseInt(req.FormValue("limit"), 10, 64)
	}
	updates, updateTypes, dates, errSelect := FindRecentChanges(currentAcc, limit)
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
				Link:    GetLink(currentAcc, upd.Message.ChatId, upd.Message.Id),
				IntLink: fmt.Sprintf("/m/%d/%d", upd.Message.ChatId, upd.Message.Id), //@TODO: link shoud be /m
				Chat: structs.ChatInfo{
					ChatId:   upd.Message.ChatId,
					ChatName: GetChatName(currentAcc, upd.Message.ChatId),
				},
			}
			if upd.Message.SenderId.MessageSenderType() == "messageSenderChat" {
			} else {
				item.From = structs.ChatInfo{ChatId: GetChatIdBySender(upd.Message.SenderId), ChatName: GetSenderName(currentAcc, upd.Message.SenderId)}
			}
			data.J = append(data.J, item)

			break
		case "updateMessageEdited":
			upd, _ := client.UnmarshalUpdateMessageEdited(rawJsonBytes)
			item := structs.JournalItem{
				T:       updateTypes[i],
				Time:    dates[i],
				Date:    FormatDateTime(dates[i]),
				Link:    GetLink(currentAcc, upd.ChatId, upd.MessageId),
				IntLink: fmt.Sprintf("/m/%d/%d", upd.ChatId, upd.MessageId),
				Chat: structs.ChatInfo{
					ChatId:   upd.ChatId,
					ChatName: GetChatName(currentAcc, upd.ChatId),
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
				Link:    GetLink(currentAcc, upd.ChatId, upd.MessageId),
				IntLink: fmt.Sprintf("/m/%d/%d", upd.ChatId, upd.MessageId),
				Chat: structs.ChatInfo{
					ChatId:   upd.ChatId,
					ChatName: GetChatName(currentAcc, upd.ChatId),
				},
			}
			m, err := FindUpdateNewMessage(currentAcc, upd.ChatId, upd.MessageId)
			if err != nil {
				item.Error = fmt.Sprintf("Message not found: %s", err)
				data.J = append(data.J, item)

				break
			}

			if m.Message.SenderId.MessageSenderType() == "messageSenderChat" {
			} else {
				item.From = structs.ChatInfo{ChatId: GetChatIdBySender(m.Message.SenderId), ChatName: GetSenderName(currentAcc, m.Message.SenderId)}
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
					ChatId:   upd.ChatId,
					ChatName: GetChatName(currentAcc, upd.ChatId),
				},
				MessageId: upd.MessageIds,
			}
			data.J = append(data.J, item)
			break
		default:
			//fc += fmt.Sprintf("[%s] Unknown update type \"%s\"<br>", FormatDateTime(dates[i]), updateTypes[i])
		}
	}
	renderTemplates(req, w, data, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/journal.tmpl`)
}

func processTdlibOptions(req *http.Request, w http.ResponseWriter) {
	actualOptions := make(map[string]structs.TdlibOption, len(tdlibOptions))
	for optionName, optionValue := range tdlibOptions[currentAcc] {
		req := client.GetOptionRequest{Name: optionName}
		res, err := tdlibClient[currentAcc].GetOption(&req)
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
			optionValue.Value = actualOption.Value
		case client.TypeOptionValueBoolean:
			actualOption := res.(*client.OptionValueBoolean)
			optionValue.Value = actualOption.Value
		case client.TypeOptionValueEmpty:
			optionValue.Value = nil
		}
		actualOptions[optionName] = optionValue
	}
	data := structs.OptionsList{T: "OptionsLists", Options: actualOptions}
	renderTemplates(req, w, data, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/tdlib_options.tmpl`)
}

func processTgActiveSessions(req *http.Request, w http.ResponseWriter) {
	sessions, err := tdlibClient[currentAcc].GetActiveSessions()
	if err != nil {
		fmt.Printf("Get sessions error: %s", err)
		return
	}
	data := structs.SessionsList{T: "Sessions", Sessions: sessions}
	if !verbose {
		data.SessionsRaw = jsonMarshalPretty(sessions)
	}

	renderTemplates(req, w, data, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/sessions_list.tmpl`)
}

func processTgSingleMessage(chatId int64, messageId int64, req *http.Request, w http.ResponseWriter) {
	upd, err := FindUpdateNewMessage(currentAcc, chatId, messageId)
	if err != nil {
		m := structs.MessageError{T: "Error", MessageId: messageId, Error: fmt.Sprintf("Error: %s", err)}
		renderTemplates(req, w, m, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/error.tmpl`)

		return
	}

	senderChatId := GetChatIdBySender(upd.Message.SenderId)
	ct := GetContentWithText(upd.Message.Content, upd.Message.ChatId)
	msg := structs.MessageInfo{
		T:             "NewMessage",
		MessageId:     upd.Message.Id,
		Date:          upd.Message.Date,
		ChatId:        upd.Message.ChatId,
		ChatName:      GetChatName(currentAcc, upd.Message.ChatId),
		SenderId:      senderChatId,
		SenderName:    GetSenderName(currentAcc, upd.Message.SenderId),
		MediaAlbumId:  int64(upd.Message.MediaAlbumId),
		SimpleText:    ct.Text,
		FormattedText: ct.FormattedText,
		Attachments:   GetContentAttachments(upd.Message.Content),
		Deleted:       IsMessageDeleted(currentAcc, upd.Message.ChatId, upd.Message.Id),
		Edited:        IsMessageEdited(currentAcc, upd.Message.ChatId, upd.Message.Id),
		ContentRaw:    nil,
	}
	chat, _ := GetChat(currentAcc, upd.Message.ChatId, false)
	res := structs.SingleMessage{
		T:       "Message",
		Message: msg,
		Edits:   make([]structs.MessageEditedInfo, 0),
		Chat:    buildChatInfoByLocalChat(chat, false),
	}

	updates, updateTypes, dates, err := FindAllMessageChanges(currentAcc, chatId, messageId)
	if err != nil {
		m := structs.MessageError{T: "Error", MessageId: messageId, Error: fmt.Sprintf("Error: %s", err)}
		renderTemplates(req, w, m, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/error.tmpl`)
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
			edit = structs.MessageEditedInfo{T: "MessageEdited"}
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

	renderTemplates(req, w, res, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/single_message.tmpl`, `templates/message.tmpl`)
}

func processTgMessagesByIds(chatId int64, req *http.Request, w http.ResponseWriter) {
	messageIds := ExplodeInt(req.FormValue("ids"))
	res := structs.ChatHistory{
		T:        "ChatHistory-filtered",
		Messages: make([]structs.MessageInfo, 0),
	}

	for _, messageId := range messageIds {
		upd, err := FindUpdateNewMessage(currentAcc, chatId, messageId)
		if err != nil {
			m := structs.MessageInfo{T: "Error", MessageId: messageId, SimpleText: fmt.Sprintf("Error: %s", err)}
			res.Messages = append(res.Messages, m)

			continue
		}

		res.Messages = append(res.Messages, parseUpdateNewMessage(upd))
	}

	renderTemplates(req, w, res, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/chat_history_filtered.tmpl`, `templates/messages_list.tmpl`, `templates/message.tmpl`)
}

func processTgChatInfo(chatId int64, req *http.Request, w http.ResponseWriter) {
	var chat interface{}
	var err error
	if chatId > 0 {
		chat, err = GetUser(currentAcc, chatId)
	} else {
		chat, err = GetChat(currentAcc, chatId, false)
	}
	if err != nil {
		fmt.Printf("Error get chat: %s\n", err)
		return
	}

	res := structs.ChatFullInfo{
		T:    "ChatFullInfo",
		Chat: chat,
	}
	var data interface{}
	if verbose {
		data = res
	} else {
		res.ChatRaw = jsonMarshalPretty(chat)
		data = res
	}
	renderTemplates(req, w, data, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/chat_info.tmpl`)
}

func processTgChatHistory(chatId int64, req *http.Request, w http.ResponseWriter) {
	deleted := false
	if req.FormValue("deleted") == "1" {
		deleted = true
	}
	limit := int64(50)
	if req.FormValue("limit") != "" {
		limit, _ = strconv.ParseInt(req.FormValue("limit"), 10, 64)
	}
	offset := int64(0)
	if req.FormValue("offset") != "" {
		offset, _ = strconv.ParseInt(req.FormValue("offset"), 10, 64)
	}

	updates, _, _, errSelect := GetChatHistory(currentAcc, chatId, limit, offset, deleted)
	if errSelect != nil {
		fmt.Printf("Error select updates: %s\n", errSelect)

		return
	}
	chat, _ := GetChat(currentAcc, chatId, false)
	res := structs.ChatHistory{
		T:          "ChatHistory",
		Chat:       buildChatInfoByLocalChat(chat, false),
		Limit:      limit,
		Offset:     offset,
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

	renderTemplates(req, w, res, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/chat_history.tmpl`, `templates/messages_list.tmpl`, `templates/message.tmpl`)
}

func processTgChatList(req *http.Request, w http.ResponseWriter) {
	refresh := false
	if req.FormValue("refresh") == "1" {
		refresh = true
	}
	var folder int32 = ClMain
	if req.FormValue("folder") != "" {
		folder64, _ := strconv.ParseInt(req.FormValue("folder"), 10, 32)
		folder = int32(folder64)
	}

	var folders []structs.ChatFolder
	folders = make([]structs.ChatFolder, 0)
	folders = append(folders, structs.ChatFolder{T: "ChatFolder", Id: ClMain, Title: "Main"})
	folders = append(folders, structs.ChatFolder{T: "ChatFolder", Id: ClArchive, Title: "Archive"})
	folders = append(folders, structs.ChatFolder{T: "ChatFolder", Id: ClCached, Title: "Cached"})
	folders = append(folders, structs.ChatFolder{T: "ChatFolder", Id: ClMy, Title: "Owned chats"})
	folders = append(folders, structs.ChatFolder{T: "ChatFolder", Id: ClNotSubscribed, Title: "Not subscribed chats"})
	for _, filter := range chatFilters[currentAcc] {
		folders = append(folders, structs.ChatFolder{T: "ChatFolder", Id: filter.Id, Title: filter.Title})
	}

	res := structs.ChatList{T: "Chat list", ChatFolders: folders, SelectedFolder: folder}
	if folder == ClCached {
		for _, chat := range localChats[currentAcc] {
			info := buildChatInfoByLocalChat(chat, true)
			res.Chats = append(res.Chats, info)
		}
	} else if folder == ClMy {
		for _, chat := range localChats[currentAcc] {
			m := client.MessageSenderUser{UserId: me[currentAcc].Id}
			req := &client.GetChatMemberRequest{ChatId: chat.Id, MemberId: &m}
			cm, err := tdlibClient[currentAcc].GetChatMember(req)
			if err != nil {
				fmt.Printf("failed to get chat member status: %d, `%s`, %s\n", chat.Id, GetChatName(currentAcc, chat.Id), err)
				continue
			}
			switch cm.Status.ChatMemberStatusType() {
			case client.TypeChatMemberStatusCreator:
				res.Chats = append(res.Chats, structs.ChatInfo{ChatId: chat.Id, ChatName: GetChatName(currentAcc, chat.Id)})
			case client.TypeChatMemberStatusAdministrator:
			case client.TypeChatMemberStatusMember:
			case client.TypeChatMemberStatusLeft:
			default:
				fmt.Printf("Unusual chat memer status: %d, `%s`, %s\n", chat.Id, GetChatName(currentAcc, chat.Id), cm.Status.ChatMemberStatusType())

			}
		}
	} else if folder == ClNotSubscribed {
		for _, chat := range localChats[currentAcc] {
			if chat.LastMessage == nil && chat.LastReadInboxMessageId == 0 {
				info := buildChatInfoByLocalChat(chat, true)
				res.Chats = append(res.Chats, info)
			}
		}
	} else if refresh {
		loadChatsList(currentAcc, folder)
		http.Redirect(w, req, fmt.Sprintf("/l?folder=%d", folder), 302)
		return
	} else {
		chatList := getSavedChats(currentAcc, folder)
		for _, chatPos := range chatList {
			chat, err := GetChat(currentAcc, chatPos.ChatId, false)
			var chatInfo structs.ChatInfo
			if err != nil {
				chatInfo = structs.ChatInfo{ChatId: chatPos.ChatId, ChatName: GetChatName(currentAcc, chatPos.ChatId), Username: "ERROR " + err.Error()}
			} else {
				chatInfo = buildChatInfoByLocalChat(chat, true)
			}

			res.Chats = append(res.Chats, chatInfo)
		}
	}

	renderTemplates(req, w, res, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/overview_table.tmpl`, `templates/chatlist.tmpl`)
}

func processTgDelete(chatId int64, req *http.Request, w http.ResponseWriter) {

	pattern := req.FormValue("pattern")
	if pattern == "" || len(pattern) < 3 {
		errorResponse(structs.WebError{T: "Invalid pattern", Error: pattern}, 503, req, w)

		return
	}
	limit := 50
	if req.FormValue("limit") != "" {
		limit64, _ := strconv.ParseInt(req.FormValue("limit"), 10, 0)
		limit = int(limit64)
	}

	var messageIds []int64
	messageIds = make([]int64, 0)
	var lastId int64 = 0
	for len(messageIds) < limit {
		req := &client.GetChatHistoryRequest{ChatId: chatId, Limit: 100, FromMessageId: lastId, Offset: 0}
		history, err := tdlibClient[currentAcc].GetChatHistory(req)
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
	ok, err := tdlibClient[currentAcc].DeleteMessages(reqDelete)
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

func processSettings(req *http.Request, w http.ResponseWriter) {
	var res structs.IgnoreLists
	if req.Method == "POST" {
		//@TODO: VALIDATE FORM DATA!! Only int acceptable as chat ID, only valid names for folders
		IgnoreChatIds := make(map[string]bool, 0)
		if _, ok := req.PostForm["ignoreChatIds"]; ok {
			for _, chatId := range req.PostForm["ignoreChatIds"] {
				if chatId == "" {
					continue
				}
				IgnoreChatIds[chatId] = true
			}
		}
		IgnoreAuthorIds := make(map[string]bool, 0)
		if _, ok := req.PostForm["ignoreAuthorIds"]; ok {
			for _, authorId := range req.PostForm["ignoreAuthorIds"] {
				if authorId == "" {
					continue
				}
				IgnoreAuthorIds[authorId] = true
			}
		}
		IgnoreFolders := make(map[string]bool, 0)
		if _, ok := req.PostForm["ignoreFolders"]; ok {
			for _, folder := range req.PostForm["ignoreFolders"] {
				if folder == "" {
					continue
				}
				IgnoreFolders[folder] = true
			}
		}
		ignoreLists[currentAcc] = structs.IgnoreLists{
			T:               "ignore_lists",
			IgnoreChatIds:   IgnoreChatIds,
			IgnoreAuthorIds: IgnoreAuthorIds,
			IgnoreFolders:   IgnoreFolders,
		}
		saveSettings(currentAcc)
		res = ignoreLists[currentAcc]
		res.T = "Settings"
		http.Redirect(w, req, "/s", 302)
		return
	} else {
		res = ignoreLists[currentAcc]
		res.T = "Settings"
	}

	renderTemplates(req, w, res, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/settings.tmpl`)
}

var st = structs.NewAccountState{}

func processAddAccount(req *http.Request, w http.ResponseWriter) {

	if currentAuthorizingAcc == nil && req.Method == "GET" {
		st = structs.NewAccountState{}
	}
	if state == nil {
		st.State = "start"
	} else if state.AuthorizationStateType() == client.TypeAuthorizationStateWaitCode {
		st.State = "code"
		st.Phone = currentAuthorizingAcc.Phone
	} else if state.AuthorizationStateType() == client.TypeAuthorizationStateWaitPassword {
		st.State = "password"
		st.Phone = currentAuthorizingAcc.Phone
	} else {
		st.State = state.AuthorizationStateType()
		st.Phone = currentAuthorizingAcc.Phone
	}

	if req.Method == "POST" {
		if currentAuthorizingAcc == nil {
			if req.FormValue("phone") != "" {
				CreateAccount(req.FormValue("phone"))
				if currentAuthorizingAcc.Status == AccStatusActive {
					st.State = "already_authorized"
					currentAuthorizingAcc = nil
				} else {
					st.State = "wait"
				}
				st.Phone = req.FormValue("phone")
			} else {
				st.State = "wtf no phone?"
			}
		} else {
			if req.FormValue("code") != "" && st.State == "code" {
				authParams <- req.FormValue("code")

				st.State = "wait"
				st.Phone = currentAuthorizingAcc.Phone
				st.Code = req.FormValue("code")
			} else if req.FormValue("password") != "" && st.State == "password" {
				authParams <- req.FormValue("password")

				st.State = "wait"
				st.Phone = currentAuthorizingAcc.Phone
				st.Code = req.FormValue("code")
				st.Password = req.FormValue("password")
			} else {
				st.State = "must refresh form without POST"
			}
		}
		http.Redirect(w, req, "/new", 302)
		return
	} else {

		renderTemplates(req, w, st, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/account_add.tmpl`)
	}
}

func processTgLink(req *http.Request, w http.ResponseWriter) {
	var link string
	if req.FormValue("link") != "" {
		link = req.FormValue("link")
	} else {
		errorResponse(structs.WebError{T: "Bad request", Error: "Invalid link"}, 400, req, w)
		return
	}

	linkInfo, LinkData, err := GetLinkInfo(currentAcc, link)
	if err != nil {
		errorResponse(structs.WebError{T: "Bad request", Error: err.Error()}, 400, req, w)
		return
	}
	respStruct := struct {
		T           string
		SourceLink  string
		LinkInfoRaw string
		LinkDataRaw string
	}{T: "Link info", SourceLink: link, LinkInfoRaw: jsonMarshalPretty(linkInfo), LinkDataRaw: jsonMarshalPretty(LinkData)}

	renderTemplates(req, w, respStruct, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/link_info.tmpl`)
}

func errorResponse(error structs.WebError, code int, req *http.Request, w http.ResponseWriter) {
	w.WriteHeader(code)
	renderTemplates(req, w, error, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/error.tmpl`)
}

func tryFile(req *http.Request, w http.ResponseWriter) bool {
	i := strings.Index(req.URL.Path, "/web/")
	var path string
	if i == -1 {
		path = "web/" + req.URL.Path
	} else if i == 0 {
		path = req.URL.Path[1:]
	} else {
		errorResponse(structs.WebError{T: "Not found", Error: "Invalid path"}, 404, req, w)

		return true
	}
	stat, err := os.Stat(path)
	if err == nil && !stat.IsDir() {
		http.ServeFile(w, req, path)

		return true
	}

	return false
}
