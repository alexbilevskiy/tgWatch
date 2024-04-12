package web

import (
	"fmt"
	"github.com/alexbilevskiy/tgWatch/pkg/libs"
	"github.com/alexbilevskiy/tgWatch/pkg/libs/tdlib"
	"github.com/alexbilevskiy/tgWatch/pkg/libs/tdlib/tdAccount"
	"github.com/alexbilevskiy/tgWatch/pkg/structs"
	"github.com/zelenin/go-tdlib/client"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type webController struct {
}

func (wc *webController) Init() {
}

func (wc *webController) processTdlibOptions(req *http.Request, w http.ResponseWriter) {
	actualOptions := make(map[string]structs.TdlibOption, len(tdlib.TdlibOptions))
	for optionName, optionValue := range tdlib.TdlibOptions {
		res, err := libs.AS.Get(currentAcc).TdApi.GetTdlibOption(optionName)
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

func (wc *webController) processTgActiveSessions(req *http.Request, w http.ResponseWriter) {
	sessions, err := libs.AS.Get(currentAcc).TdApi.GetActiveSessions()
	if err != nil {
		fmt.Printf("Get sessions error: %s", err)
		return
	}
	data := structs.SessionsList{T: "Sessions", Sessions: sessions}
	if !verbose {
		data.SessionsRaw = string(libs.JsonMarshalPretty(sessions))
	}

	renderTemplates(req, w, data, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/sessions_list.gohtml`)
}

func (wc *webController) processTgSingleMessage(chatId int64, messageId int64, req *http.Request, w http.ResponseWriter) {

	message, err := libs.AS.Get(currentAcc).TdApi.GetMessage(chatId, messageId)
	if err != nil {
		m := structs.MessageError{T: "Error", MessageId: messageId, Error: fmt.Sprintf("Error: %s", err)}
		renderTemplates(req, w, m, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/error.gohtml`)

		return
	}

	senderChatId := tdlib.GetChatIdBySender(message.SenderId)
	ct := tdlib.GetContentWithText(message.Content, message.ChatId)
	msg := structs.MessageInfo{
		T:             "NewMessage",
		MessageId:     message.Id,
		Date:          message.Date,
		ChatId:        message.ChatId,
		ChatName:      libs.AS.Get(currentAcc).TdApi.GetChatName(message.ChatId),
		SenderId:      senderChatId,
		SenderName:    libs.AS.Get(currentAcc).TdApi.GetSenderName(message.SenderId),
		MediaAlbumId:  int64(message.MediaAlbumId),
		SimpleText:    ct.Text,
		FormattedText: ct.FormattedText,
		Attachments:   tdlib.GetContentAttachments(message.Content),
		Edited:        message.EditDate != 0,
		ContentRaw:    message,
	}
	chat, _ := libs.AS.Get(currentAcc).TdApi.GetChat(message.ChatId, false)
	res := structs.SingleMessage{
		T:       "Message",
		Message: msg,
		Chat:    buildChatInfoByLocalChat(chat),
	}

	renderTemplates(req, w, res, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/single_message.gohtml`, `templates/message.gohtml`)
}

func (wc *webController) processTgMessagesByIds(chatId int64, req *http.Request, w http.ResponseWriter) {
	messageIds := libs.ExplodeInt(req.FormValue("ids"))
	res := structs.ChatHistoryOnline{
		T:        "ChatHistory-filtered",
		Messages: make([]structs.MessageInfo, 0),
	}
	chat, err := libs.AS.Get(currentAcc).TdApi.GetChat(chatId, false)
	if err != nil {

	} else {
		res.Chat = buildChatInfoByLocalChat(chat)
	}

	for _, messageId := range messageIds {
		message, err := libs.AS.Get(currentAcc).TdApi.GetMessage(chatId, messageId)
		if err != nil {
			m := structs.MessageInfo{T: "Error", MessageId: messageId, SimpleText: fmt.Sprintf("Error: %s", err)}
			res.Messages = append(res.Messages, m)

			continue
		}

		res.Messages = append(res.Messages, parseMessage(message))
	}

	renderTemplates(req, w, res, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/chat_history_filtered.gohtml`, `templates/messages_list.gohtml`, `templates/message.gohtml`)
}

func (wc *webController) processTgChatInfo(chatId int64, req *http.Request, w http.ResponseWriter) {
	var chat *client.Chat
	var err error
	chat, err = libs.AS.Get(currentAcc).TdApi.GetChat(chatId, false)
	if err != nil {
		fmt.Printf("Error get chat: %s\n", err)
		return
	}

	res := structs.ChatFullInfo{
		T:    "ChatFullInfo",
		Chat: buildChatInfoByLocalChat(chat),
	}
	var data interface{}
	if verbose {
		data = res
	} else {
		res.ChatRaw = string(libs.JsonMarshalPretty(chat))
		data = res
	}
	renderTemplates(req, w, data, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/chat_info.gohtml`)
}

func (wc *webController) processTgChatHistoryOnline(chatId int64, req *http.Request, w http.ResponseWriter) {
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

	messages, err := libs.AS.Get(currentAcc).TdApi.LoadChatHistory(chatId, fromMessageId, offset)
	if err != nil {
		log.Printf("error load history: %s", err.Error())

		return
	}
	chat, _ := libs.AS.Get(currentAcc).TdApi.GetChat(chatId, false)
	//@TODO: crashes if history is empty
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

func (wc *webController) processTgChatList(req *http.Request, w http.ResponseWriter) {
	refresh := false
	if req.FormValue("refresh") == "1" {
		refresh = true
	}
	var folder int32 = tdlib.ClMain
	if req.FormValue("folder") != "" {
		folder64, _ := strconv.ParseInt(req.FormValue("folder"), 10, 32)
		folder = int32(folder64)
	}
	var groupsInCommonUserId int64
	if req.FormValue("groups_in_common_userid") != "" {
		groupsInCommonUserId, _ = strconv.ParseInt(req.FormValue("groups_in_common_userid"), 10, 64)
	}

	var folders []structs.ChatFolder
	folders = make([]structs.ChatFolder, 0)
	folders = append(folders, structs.ChatFolder{T: "ChatFolder", Id: tdlib.ClMain, Title: "Main"})
	folders = append(folders, structs.ChatFolder{T: "ChatFolder", Id: tdlib.ClArchive, Title: "Archive"})
	folders = append(folders, structs.ChatFolder{T: "ChatFolder", Id: tdlib.ClCached, Title: "Cached"})
	folders = append(folders, structs.ChatFolder{T: "ChatFolder", Id: tdlib.ClOwned, Title: "Owned chats"})
	folders = append(folders, structs.ChatFolder{T: "ChatFolder", Id: tdlib.ClNotSubscribed, Title: "Not subscribed chats"})
	folders = append(folders, structs.ChatFolder{T: "ChatFolder", Id: tdlib.ClNotAssigned, Title: "Chats not in any folder"})
	for _, filter := range libs.AS.Get(currentAcc).TdApi.GetChatFolders() {
		folders = append(folders, structs.ChatFolder{T: "ChatFolder", Id: filter.Id, Title: filter.Title})
	}

	res := structs.ChatList{T: "Chat list", ChatFolders: folders, SelectedFolder: folder}
	if folder == tdlib.ClCached {
		for _, chat := range libs.AS.Get(currentAcc).TdApi.GetLocalChats() {
			info := buildChatInfoByLocalChat(chat)
			res.Chats = append(res.Chats, info)
		}
	} else if folder == tdlib.ClOwned {
		for _, chat := range libs.AS.Get(currentAcc).TdApi.GetLocalChats() {
			cm, err := libs.AS.Get(currentAcc).TdApi.GetChatMember(chat.Id)
			if err != nil && err.Error() != "400 CHANNEL_PRIVATE" {
				fmt.Printf("failed to get chat member status: %d, `%s`, %s\n", chat.Id, libs.AS.Get(currentAcc).TdApi.GetChatName(chat.Id), err)
				continue
			}
			switch cm.Status.ChatMemberStatusType() {
			case client.TypeChatMemberStatusCreator:
				res.Chats = append(res.Chats, structs.ChatInfo{ChatId: chat.Id, ChatName: libs.AS.Get(currentAcc).TdApi.GetChatName(chat.Id)})
			case client.TypeChatMemberStatusAdministrator:
			case client.TypeChatMemberStatusMember:
			case client.TypeChatMemberStatusLeft:
			case client.TypeChatMemberStatusRestricted:
				//@todo: print restrictions
			default:
				fmt.Printf("Unusual chat memer status: %d, `%s`, %s\n", chat.Id, libs.AS.Get(currentAcc).TdApi.GetChatName(chat.Id), cm.Status.ChatMemberStatusType())

			}
		}
	} else if folder == tdlib.ClNotSubscribed {
		for _, chat := range libs.AS.Get(currentAcc).TdApi.GetLocalChats() {
			if chat.LastMessage == nil && chat.LastReadInboxMessageId == 0 {
				info := buildChatInfoByLocalChat(chat)
				res.Chats = append(res.Chats, info)
			}
		}
	} else if folder == tdlib.ClNotAssigned {
		for _, chat := range libs.AS.Get(currentAcc).TdApi.GetLocalChats() {
			saved := false
			for _, filter := range libs.AS.Get(currentAcc).TdApi.GetChatFolders() {
				savedChats := libs.AS.Get(currentAcc).TdApi.GetStorage().GetSavedChats(filter.Id)
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
		libs.AS.Get(currentAcc).TdApi.LoadChatsList(folder)
		http.Redirect(w, req, fmt.Sprintf("/l?folder=%d", folder), 302)
		return
	} else if groupsInCommonUserId != 0 {
		if req.Method == "POST" {

			var addToFolder int32
			if req.FormValue("add_to_folder") != "" {
				folder64, _ := strconv.ParseInt(req.FormValue("add_to_folder"), 10, 32)
				addToFolder = int32(folder64)
			}

			addChatsToFolder := make([]int64, 0)
			if _, ok := req.PostForm["chats"]; ok {
				for _, chatIdStr := range req.PostForm["chats"] {
					if chatIdStr == "" {
						continue
					}
					chatId, _ := strconv.ParseInt(chatIdStr, 10, 64)
					addChatsToFolder = append(addChatsToFolder, chatId)
				}
			}
			if len(addChatsToFolder) > 0 {
				//@TODO: errors validation
				libs.AS.Get(currentAcc).TdApi.AddChatsToFolder(addChatsToFolder, addToFolder)
			}
			http.Redirect(w, req, fmt.Sprintf("/l?groups_in_common_userid=%d", groupsInCommonUserId), 302)
		}
		partnerChat, _ := libs.AS.Get(currentAcc).TdApi.GetChat(groupsInCommonUserId, false)
		res.PartnerChat = buildChatInfoByLocalChat(partnerChat)
		chats, err := libs.AS.Get(currentAcc).TdApi.GetGroupsInCommon(groupsInCommonUserId)
		if err != nil {
			log.Printf("failed to get groups in common: %d, `%s`, %s", groupsInCommonUserId, libs.AS.Get(currentAcc).TdApi.GetChatName(groupsInCommonUserId), err)
		}
		for _, chatId := range chats.ChatIds {
			chat, err := libs.AS.Get(currentAcc).TdApi.GetChat(chatId, true)
			var chatInfo structs.ChatInfo
			if err != nil {
				chatInfo = structs.ChatInfo{ChatId: chatId, ChatName: libs.AS.Get(currentAcc).TdApi.GetChatName(chatId), Username: "ERROR " + err.Error()}
			} else {
				chatInfo = buildChatInfoByLocalChat(chat)
			}

			res.Chats = append(res.Chats, chatInfo)
		}
	} else {
		chatList := libs.AS.Get(currentAcc).TdApi.GetStorage().GetSavedChats(folder)
		for _, chatPos := range chatList {
			chat, err := libs.AS.Get(currentAcc).TdApi.GetChat(chatPos.ChatId, true)
			var chatInfo structs.ChatInfo
			if err != nil {
				chatInfo = structs.ChatInfo{ChatId: chatPos.ChatId, ChatName: libs.AS.Get(currentAcc).TdApi.GetChatName(chatPos.ChatId), Username: "ERROR " + err.Error()}
			} else {
				chatInfo = buildChatInfoByLocalChat(chat)
			}

			res.Chats = append(res.Chats, chatInfo)
		}
	}

	renderTemplates(req, w, res, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/overview_table.gohtml`, `templates/chatlist.gohtml`)
}

func (wc *webController) processTgDelete(chatId int64, req *http.Request, w http.ResponseWriter) {

	pattern := req.FormValue("pattern")
	if pattern == "" || len(pattern) < 3 {
		wc.errorResponse(structs.WebError{T: "Invalid pattern", Error: pattern}, 503, req, w)

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
		history, err := libs.AS.Get(currentAcc).TdApi.GetChatHistory(chatId, lastId)
		if err != nil {
			data := []byte(fmt.Sprintf("Error get chat %d history: %s", chatId, err))
			w.Write(data)
			return
		}
		fmt.Printf("Received history of %d messages from chat %d\n", history.TotalCount, chatId)
		noMore := true
		for _, message := range history.Messages {
			lastId = message.Id
			content := tdlib.GetContentWithText(message.Content, message.ChatId).Text
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
	ok, err := libs.AS.Get(currentAcc).TdApi.DeleteMessages(chatId, messageIds)
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

func (wc *webController) processSettings(req *http.Request, w http.ResponseWriter) {
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
		il := structs.IgnoreLists{
			T:               "ignore_lists",
			IgnoreChatIds:   IgnoreChatIds,
			IgnoreAuthorIds: IgnoreAuthorIds,
			IgnoreFolders:   IgnoreFolders,
		}
		libs.AS.Get(currentAcc).TdApi.GetStorage().SaveSettings(il)
		res = libs.AS.Get(currentAcc).TdApi.GetStorage().GetSettings()
		res.T = "Settings"
		http.Redirect(w, req, "/s", 302)
		return
	} else {
		res = libs.AS.Get(currentAcc).TdApi.GetStorage().GetSettings()
		res.T = "Settings"
	}

	renderTemplates(req, w, res, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/settings.gohtml`)
}

var st = structs.NewAccountState{}

func (wc *webController) processAddAccount(req *http.Request, w http.ResponseWriter) {

	if tdAccount.CurrentAuthorizingAcc == nil && req.Method == "GET" {
		st = structs.NewAccountState{}
	}
	if tdAccount.AuthorizerState == nil {
		st.State = "start"
	} else if tdAccount.AuthorizerState.AuthorizationStateType() == client.TypeAuthorizationStateWaitCode {
		st.State = "code"
		st.Phone = tdAccount.CurrentAuthorizingAcc.Phone
	} else if tdAccount.AuthorizerState.AuthorizationStateType() == client.TypeAuthorizationStateWaitPassword {
		st.State = "password"
		st.Phone = tdAccount.CurrentAuthorizingAcc.Phone
	} else {
		st.State = tdAccount.AuthorizerState.AuthorizationStateType()
		st.Phone = tdAccount.CurrentAuthorizingAcc.Phone
	}

	if req.Method == "POST" {
		if tdAccount.CurrentAuthorizingAcc == nil {
			if req.FormValue("phone") != "" {
				tdAccount.CreateAccount(req.FormValue("phone"))
				if tdAccount.CurrentAuthorizingAcc.Status == tdlib.AccStatusActive {
					st.State = "already_authorized"
					tdAccount.CurrentAuthorizingAcc = nil
				} else {
					st.State = "wait"
				}
				st.Phone = req.FormValue("phone")
			} else {
				st.State = "wtf no phone?"
			}
		} else {
			if req.FormValue("code") != "" && st.State == "code" {
				tdAccount.AuthParams <- req.FormValue("code")

				st.State = "wait"
				st.Phone = tdAccount.CurrentAuthorizingAcc.Phone
				st.Code = req.FormValue("code")
			} else if req.FormValue("password") != "" && st.State == "password" {
				tdAccount.AuthParams <- req.FormValue("password")

				st.State = "wait"
				st.Phone = tdAccount.CurrentAuthorizingAcc.Phone
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

func (wc *webController) processTgLink(req *http.Request, w http.ResponseWriter) {
	var link string
	if req.FormValue("link") != "" {
		link = req.FormValue("link")
	} else {
		wc.errorResponse(structs.WebError{T: "Bad request", Error: "Invalid link"}, 400, req, w)
		return
	}

	linkInfo, LinkData, err := libs.AS.Get(currentAcc).TdApi.GetLinkInfo(link)
	if err != nil {
		wc.errorResponse(structs.WebError{T: "Bad request", Error: err.Error()}, 400, req, w)
		return
	}
	respStruct := struct {
		T           string
		SourceLink  string
		LinkInfoRaw string
		LinkDataRaw string
	}{T: "Link info", SourceLink: link, LinkInfoRaw: string(libs.JsonMarshalPretty(linkInfo)), LinkDataRaw: string(libs.JsonMarshalPretty(LinkData))}

	renderTemplates(req, w, respStruct, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/link_info.gohtml`)
}

func (wc *webController) errorResponse(error structs.WebError, code int, req *http.Request, w http.ResponseWriter) {
	w.WriteHeader(code)
	renderTemplates(req, w, error, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/error.gohtml`)
}
