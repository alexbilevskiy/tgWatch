package libs

import (
	"fmt"
	"github.com/alexbilevskiy/tgWatch/pkg/structs"
	"github.com/zelenin/go-tdlib/client"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func processTdlibOptions(req *http.Request, w http.ResponseWriter) {
	actualOptions := make(map[string]structs.TdlibOption, len(tdlibOptions))
	for optionName, optionValue := range tdlibOptions[currentAcc] {
		req := client.GetOptionRequest{Name: optionName}
		res, err := tdlibClient[currentAcc].GetOptionAsync(&req)
		if err != nil {
			log.Printf("Failed to get option %s: %s", optionName, err)
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
	renderTemplates(req, w, data, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/tdlib_options.gohtml`)
}

func processTgActiveSessions(req *http.Request, w http.ResponseWriter) {
	sessions, err := tdlibClient[currentAcc].GetActiveSessions()
	if err != nil {
		fmt.Printf("Get sessions error: %s", err)
		return
	}
	data := structs.SessionsList{T: "Sessions", Sessions: sessions}
	if !verbose {
		data.SessionsRaw = string(jsonMarshalPretty(sessions))
	}

	renderTemplates(req, w, data, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/sessions_list.gohtml`)
}

func processTgSingleMessage(chatId int64, messageId int64, req *http.Request, w http.ResponseWriter) {

	message, err := GetMessage(currentAcc, chatId, messageId)
	if err != nil {
		m := structs.MessageError{T: "Error", MessageId: messageId, Error: fmt.Sprintf("Error: %s", err)}
		renderTemplates(req, w, m, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/error.gohtml`)

		return
	}

	senderChatId := GetChatIdBySender(message.SenderId)
	ct := GetContentWithText(message.Content, message.ChatId)
	msg := structs.MessageInfo{
		T:             "NewMessage",
		MessageId:     message.Id,
		Date:          message.Date,
		ChatId:        message.ChatId,
		ChatName:      GetChatName(currentAcc, message.ChatId),
		SenderId:      senderChatId,
		SenderName:    GetSenderName(currentAcc, message.SenderId),
		MediaAlbumId:  int64(message.MediaAlbumId),
		SimpleText:    ct.Text,
		FormattedText: ct.FormattedText,
		Attachments:   GetContentAttachments(message.Content),
		Edited:        message.EditDate != 0,
		ContentRaw:    message,
	}
	chat, _ := GetChat(currentAcc, message.ChatId, false)
	res := structs.SingleMessage{
		T:       "Message",
		Message: msg,
		Chat:    buildChatInfoByLocalChat(chat),
	}

	renderTemplates(req, w, res, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/single_message.gohtml`, `templates/message.gohtml`)
}

func processTgMessagesByIds(chatId int64, req *http.Request, w http.ResponseWriter) {
	messageIds := ExplodeInt(req.FormValue("ids"))
	res := structs.ChatHistoryOnline{
		T:        "ChatHistory-filtered",
		Messages: make([]structs.MessageInfo, 0),
	}
	chat, err := GetChat(currentAcc, chatId, false)
	if err != nil {

	} else {
		res.Chat = buildChatInfoByLocalChat(chat)
	}

	for _, messageId := range messageIds {
		message, err := GetMessage(currentAcc, chatId, messageId)
		if err != nil {
			m := structs.MessageInfo{T: "Error", MessageId: messageId, SimpleText: fmt.Sprintf("Error: %s", err)}
			res.Messages = append(res.Messages, m)

			continue
		}

		res.Messages = append(res.Messages, parseMessage(message))
	}

	renderTemplates(req, w, res, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/chat_history_filtered.gohtml`, `templates/messages_list.gohtml`, `templates/message.gohtml`)
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
		res.ChatRaw = string(jsonMarshalPretty(chat))
		data = res
	}
	renderTemplates(req, w, data, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/chat_info.gohtml`)
}

func processTgChatHistoryOnline(chatId int64, req *http.Request, w http.ResponseWriter) {
	var fromMessageId int64 = 0
	var offset int32 = 0
	var err error
	if req.FormValue("from_message_id") != "" {
		fromMessageId, err = strconv.ParseInt(req.FormValue("from_message_id"), 10, 64)
		if err != nil {
			log.Printf("failed to parse from_message_id: %s", err.Error())
			return
		}
	}
	if req.FormValue("offset") != "" {
		offset64, err := strconv.ParseInt(req.FormValue("offset"), 10, 32)
		if err != nil {
			log.Printf("failed to parse from_message_id fromMessageId: %s", err.Error())
			return
		}
		offset = int32(offset64)
	}

	messages, err := LoadChatHistory(currentAcc, chatId, fromMessageId, offset)
	if err != nil {
		log.Printf("error load history: %s", err.Error())

		return
	}
	chat, _ := GetChat(currentAcc, chatId, false)
	res := structs.ChatHistoryOnline{
		T:    "ChatHistory",
		Chat: buildChatInfoByLocalChat(chat),
		//wicked!
		FirstMessageId: messages.Messages[0].Id,
		LastMessageId:  messages.Messages[len(messages.Messages)-1].Id,
		NextOffset:     -50,
		PrevOffset:     0,
	}

	for _, message := range messages.Messages {
		messageInfo := parseMessage(message)
		//hack to reverse, orig was: res.Messages = append(res.Messages, messageInfo)
		res.Messages = append([]structs.MessageInfo{messageInfo}, res.Messages...)
	}

	renderTemplates(req, w, res, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/chat_history_online.gohtml`, `templates/messages_list.gohtml`, `templates/message.gohtml`)
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
	folders = append(folders, structs.ChatFolder{T: "ChatFolder", Id: ClOwned, Title: "Owned chats"})
	folders = append(folders, structs.ChatFolder{T: "ChatFolder", Id: ClNotSubscribed, Title: "Not subscribed chats"})
	folders = append(folders, structs.ChatFolder{T: "ChatFolder", Id: ClNotAssigned, Title: "Chats not in any folder"})
	for _, filter := range chatFolders[currentAcc] {
		folders = append(folders, structs.ChatFolder{T: "ChatFolder", Id: filter.Id, Title: filter.Title})
	}

	res := structs.ChatList{T: "Chat list", ChatFolders: folders, SelectedFolder: folder}
	if folder == ClCached {
		for _, chat := range localChats[currentAcc] {
			info := buildChatInfoByLocalChat(chat)
			res.Chats = append(res.Chats, info)
		}
	} else if folder == ClOwned {
		for _, chat := range localChats[currentAcc] {
			m := client.MessageSenderUser{UserId: me[currentAcc].Id}
			req := &client.GetChatMemberRequest{ChatId: chat.Id, MemberId: &m}
			cm, err := tdlibClient[currentAcc].GetChatMember(req)
			if err != nil && err.Error() != "400 CHANNEL_PRIVATE" {
				fmt.Printf("failed to get chat member status: %d, `%s`, %s\n", chat.Id, GetChatName(currentAcc, chat.Id), err)
				continue
			}
			switch cm.Status.ChatMemberStatusType() {
			case client.TypeChatMemberStatusCreator:
				res.Chats = append(res.Chats, structs.ChatInfo{ChatId: chat.Id, ChatName: GetChatName(currentAcc, chat.Id)})
			case client.TypeChatMemberStatusAdministrator:
			case client.TypeChatMemberStatusMember:
			case client.TypeChatMemberStatusLeft:
			case client.TypeChatMemberStatusRestricted:
				//@todo: print restrictions
			default:
				fmt.Printf("Unusual chat memer status: %d, `%s`, %s\n", chat.Id, GetChatName(currentAcc, chat.Id), cm.Status.ChatMemberStatusType())

			}
		}
	} else if folder == ClNotSubscribed {
		for _, chat := range localChats[currentAcc] {
			if chat.LastMessage == nil && chat.LastReadInboxMessageId == 0 {
				info := buildChatInfoByLocalChat(chat)
				res.Chats = append(res.Chats, info)
			}
		}
	} else if folder == ClNotAssigned {
		for _, chat := range localChats[currentAcc] {
			saved := false
			for _, filter := range chatFolders[currentAcc] {
				savedChats := getSavedChats(currentAcc, filter.Id)
				for _, pos := range savedChats {
					if pos.ChatId == chat.Id {
						saved = true
					}
				}
			}
			if !saved {
				if chat.Type.ChatTypeType() == client.TypeChatTypePrivate {
					continue
				}
				info := buildChatInfoByLocalChat(chat)
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
			chat, err := GetChat(currentAcc, chatPos.ChatId, true)
			var chatInfo structs.ChatInfo
			if err != nil {
				chatInfo = structs.ChatInfo{ChatId: chatPos.ChatId, ChatName: GetChatName(currentAcc, chatPos.ChatId), Username: "ERROR " + err.Error()}
			} else {
				chatInfo = buildChatInfoByLocalChat(chat)
			}

			res.Chats = append(res.Chats, chatInfo)
		}
	}

	renderTemplates(req, w, res, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/overview_table.gohtml`, `templates/chatlist.gohtml`)
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

	renderTemplates(req, w, res, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/settings.gohtml`)
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

		renderTemplates(req, w, st, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/account_add.gohtml`)
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
	}{T: "Link info", SourceLink: link, LinkInfoRaw: string(jsonMarshalPretty(linkInfo)), LinkDataRaw: string(jsonMarshalPretty(LinkData))}

	renderTemplates(req, w, respStruct, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/link_info.gohtml`)
}

func errorResponse(error structs.WebError, code int, req *http.Request, w http.ResponseWriter) {
	w.WriteHeader(code)
	renderTemplates(req, w, error, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/error.gohtml`)
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
