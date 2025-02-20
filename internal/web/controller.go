package web

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/alexbilevskiy/tgWatch/internal/account"
	"github.com/alexbilevskiy/tgWatch/internal/config"
	"github.com/alexbilevskiy/tgWatch/internal/consts"
	"github.com/alexbilevskiy/tgWatch/internal/helpers"
	"github.com/alexbilevskiy/tgWatch/internal/tdlib"
	"github.com/alexbilevskiy/tgWatch/internal/tdlib/tdAccount"
	"github.com/zelenin/go-tdlib/client"
)

type newAccountState struct {
	T        string
	Phone    string
	Code     string
	Password string
	State    string
}

type webController struct {
	cfg *config.Config
	st  newAccountState
}

func newWebController(cfg *config.Config) *webController {
	return &webController{cfg: cfg}
}

func (wc *webController) processRoot(w http.ResponseWriter, r *http.Request) {
	renderTemplates(r, w, nil, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/index.gohtml`)
}

func (wc *webController) processTdlibOptions(w http.ResponseWriter, req *http.Request) {
	actualOptions := make(map[string]tdlib.TdlibOption, len(tdlib.TdlibOptions))
	for optionName, optionValue := range tdlib.TdlibOptions {
		res, err := account.AS.Get(req.Context().Value("current_acc").(int64)).TdApi.GetTdlibOption(optionName)
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
	data := OptionsList{T: "OptionsLists", Options: actualOptions}
	renderTemplates(req, w, data, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/tdlib_options.gohtml`)
}

func (wc *webController) processTgActiveSessions(w http.ResponseWriter, req *http.Request) {
	sessions, err := account.AS.Get(req.Context().Value("current_acc").(int64)).TdApi.GetActiveSessions()
	if err != nil {
		fmt.Printf("Get sessions error: %s", err)
		return
	}
	data := SessionsList{T: "Sessions", Sessions: sessions}
	if !req.Context().Value("verbose").(bool) {
		data.SessionsRaw = string(helpers.JsonMarshalPretty(sessions))
	}

	renderTemplates(req, w, data, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/sessions_list.gohtml`)
}

func (wc *webController) processTgSingleMessage(w http.ResponseWriter, req *http.Request) {
	chatId, _ := strconv.ParseInt(req.PathValue("chat_id"), 10, 64)
	messageId, _ := strconv.ParseInt(req.PathValue("message_id"), 10, 64)

	currentAcc := req.Context().Value("current_acc").(int64)
	message, err := account.AS.Get(currentAcc).TdApi.GetMessage(chatId, messageId)
	if err != nil {
		m := MessageError{T: "Error", MessageId: messageId, Error: fmt.Sprintf("Error: %s", err)}
		renderTemplates(req, w, m, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/error.gohtml`)

		return
	}

	senderChatId := tdlib.GetChatIdBySender(message.SenderId)
	ct := GetContentWithText(message.Content, message.ChatId)
	msg := MessageInfo{
		T:             "NewMessage",
		MessageId:     message.Id,
		Date:          message.Date,
		ChatId:        message.ChatId,
		ChatName:      account.AS.Get(currentAcc).TdApi.GetChatName(message.ChatId),
		SenderId:      senderChatId,
		SenderName:    account.AS.Get(currentAcc).TdApi.GetSenderName(message.SenderId),
		MediaAlbumId:  int64(message.MediaAlbumId),
		SimpleText:    ct.Text,
		FormattedText: ct.FormattedText,
		Attachments:   GetContentAttachments(message.Content),
		Edited:        message.EditDate != 0,
		ContentRaw:    message,
	}
	chat, _ := account.AS.Get(currentAcc).TdApi.GetChat(message.ChatId, false)
	res := SingleMessage{
		T:       "Message",
		Message: msg,
		Chat:    buildChatInfoByLocalChat(req.Context(), chat),
	}

	renderTemplates(req, w, res, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/single_message.gohtml`, `templates/message.gohtml`)
}

func (wc *webController) processTgMessagesByIds(chatId int64, req *http.Request, w http.ResponseWriter) {
	messageIds := helpers.ExplodeInt(req.FormValue("ids"))
	res := ChatHistoryOnline{
		T:        "ChatHistory-filtered",
		Messages: make([]MessageInfo, 0),
	}
	currentAcc := req.Context().Value("current_acc").(int64)
	chat, err := account.AS.Get(currentAcc).TdApi.GetChat(chatId, false)
	if err != nil {

	} else {
		res.Chat = buildChatInfoByLocalChat(req.Context(), chat)
	}

	for _, messageId := range messageIds {
		message, err := account.AS.Get(currentAcc).TdApi.GetMessage(chatId, messageId)
		if err != nil {
			m := MessageInfo{T: "Error", MessageId: messageId, SimpleText: fmt.Sprintf("Error: %s", err)}
			res.Messages = append(res.Messages, m)

			continue
		}

		res.Messages = append(res.Messages, parseMessage(message, currentAcc, req.Context().Value("verbose").(bool)))
	}

	renderTemplates(req, w, res, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/chat_history_filtered.gohtml`, `templates/messages_list.gohtml`, `templates/message.gohtml`)
}

func (wc *webController) processTgChatInfo(w http.ResponseWriter, req *http.Request) {
	chatId, _ := strconv.ParseInt(req.PathValue("chat_id"), 10, 64)
	var chat *client.Chat
	var err error
	chat, err = account.AS.Get(req.Context().Value("current_acc").(int64)).TdApi.GetChat(chatId, false)
	if err != nil {
		fmt.Printf("Error get chat: %s\n", err)
		return
	}

	res := ChatFullInfo{
		T:    "ChatFullInfo",
		Chat: buildChatInfoByLocalChat(req.Context(), chat),
	}
	var data interface{}
	if req.Context().Value("verbose").(bool) {
		data = res
	} else {
		res.ChatRaw = string(helpers.JsonMarshalPretty(chat))
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
	currentAcc := req.Context().Value("current_acc").(int64)
	messages, err := account.AS.Get(currentAcc).TdApi.LoadChatHistory(chatId, fromMessageId, offset)
	if err != nil {
		log.Printf("error load history: %s", err.Error())
		errorResponse(WebError{T: "No messages", Error: err.Error()}, http.StatusBadRequest, req, w)

		return
	}
	chat, _ := account.AS.Get(currentAcc).TdApi.GetChat(chatId, false)
	if len(messages.Messages) == 0 {
		errorResponse(WebError{T: "No messages", Error: "no saved messages"}, http.StatusBadRequest, req, w)
		return
	}
	res := ChatHistoryOnline{
		T:    "ChatHistory",
		Chat: buildChatInfoByLocalChat(req.Context(), chat),
		//wicked!
		FirstMessageId: messages.Messages[0].Id,
		LastMessageId:  messages.Messages[len(messages.Messages)-1].Id,
		NextOffset:     -50,
		PrevOffset:     0,
	}

	for _, message := range messages.Messages {
		messageInfo := parseMessage(message, currentAcc, req.Context().Value("verbose").(bool))
		//hack to reverse, orig was: res.Messages = append(res.Messages, messageInfo)
		res.Messages = append([]MessageInfo{messageInfo}, res.Messages...)
	}

	renderTemplates(req, w, res, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/chat_history_online.gohtml`, `templates/messages_list.gohtml`, `templates/message.gohtml`)
}

func (wc *webController) processTgChatList(w http.ResponseWriter, req *http.Request) {
	refresh := false
	if req.FormValue("refresh") == "1" {
		refresh = true
	}
	var folder int32 = consts.ClMain
	if req.FormValue("folder") != "" {
		folder64, _ := strconv.ParseInt(req.FormValue("folder"), 10, 32)
		folder = int32(folder64)
	}
	var groupsInCommonUserId int64
	if req.FormValue("groups_in_common_userid") != "" {
		groupsInCommonUserId, _ = strconv.ParseInt(req.FormValue("groups_in_common_userid"), 10, 64)
	}
	currentAcc := req.Context().Value("current_acc").(int64)

	var folders []ChatFolder
	folders = make([]ChatFolder, 0)
	folders = append(folders, ChatFolder{T: "ChatFolder", Id: consts.ClMain, Title: "Main"})
	folders = append(folders, ChatFolder{T: "ChatFolder", Id: consts.ClArchive, Title: "Archive"})
	folders = append(folders, ChatFolder{T: "ChatFolder", Id: consts.ClCached, Title: "Cached"})
	folders = append(folders, ChatFolder{T: "ChatFolder", Id: consts.ClOwned, Title: "Owned chats"})
	folders = append(folders, ChatFolder{T: "ChatFolder", Id: consts.ClNotSubscribed, Title: "Not subscribed chats"})
	folders = append(folders, ChatFolder{T: "ChatFolder", Id: consts.ClNotAssigned, Title: "Chats not in any folder"})
	for _, filter := range account.AS.Get(currentAcc).TdApi.GetChatFolders() {
		folders = append(folders, ChatFolder{T: "ChatFolder", Id: filter.Id, Title: filter.Title})
	}

	res := ChatList{T: "Chat list", ChatFolders: folders, SelectedFolder: folder}
	if folder == consts.ClCached {
		for _, chat := range account.AS.Get(currentAcc).TdApi.GetLocalChats() {
			info := buildChatInfoByLocalChat(req.Context(), chat)
			res.Chats = append(res.Chats, info)
		}
	} else if folder == consts.ClOwned {
		for _, chat := range account.AS.Get(currentAcc).TdApi.GetLocalChats() {
			cm, err := account.AS.Get(currentAcc).TdApi.GetChatMember(chat.Id)
			if err != nil && err.Error() != "400 CHANNEL_PRIVATE" {
				fmt.Printf("failed to get chat member status: %d, `%s`, %s\n", chat.Id, account.AS.Get(currentAcc).TdApi.GetChatName(chat.Id), err)
				continue
			}
			switch cm.Status.ChatMemberStatusType() {
			case client.TypeChatMemberStatusCreator:
				res.Chats = append(res.Chats, ChatInfo{ChatId: chat.Id, ChatName: account.AS.Get(currentAcc).TdApi.GetChatName(chat.Id)})
			case client.TypeChatMemberStatusAdministrator:
			case client.TypeChatMemberStatusMember:
			case client.TypeChatMemberStatusLeft:
			case client.TypeChatMemberStatusRestricted:
				//@todo: print restrictions
			default:
				fmt.Printf("Unusual chat memer status: %d, `%s`, %s\n", chat.Id, account.AS.Get(currentAcc).TdApi.GetChatName(chat.Id), cm.Status.ChatMemberStatusType())

			}
		}
	} else if folder == consts.ClNotSubscribed {
		for _, chat := range account.AS.Get(currentAcc).TdApi.GetLocalChats() {
			if chat.LastMessage == nil && chat.LastReadInboxMessageId == 0 {
				info := buildChatInfoByLocalChat(req.Context(), chat)
				res.Chats = append(res.Chats, info)
			}
		}
	} else if folder == consts.ClNotAssigned {
		for _, chat := range account.AS.Get(currentAcc).TdApi.GetLocalChats() {
			saved := false
			for _, filter := range account.AS.Get(currentAcc).TdApi.GetChatFolders() {
				savedChats := account.AS.Get(currentAcc).TdApi.GetStorage().GetSavedChats(filter.Id)
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
				info := buildChatInfoByLocalChat(req.Context(), chat)
				res.Chats = append(res.Chats, info)
			}
		}
	} else if refresh {
		account.AS.Get(currentAcc).TdApi.LoadChatsList(folder)
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
				account.AS.Get(currentAcc).TdApi.AddChatsToFolder(addChatsToFolder, addToFolder)
			}
			http.Redirect(w, req, fmt.Sprintf("/l?groups_in_common_userid=%d", groupsInCommonUserId), 302)
		}
		partnerChat, _ := account.AS.Get(currentAcc).TdApi.GetChat(groupsInCommonUserId, false)
		res.PartnerChat = buildChatInfoByLocalChat(req.Context(), partnerChat)
		chats, err := account.AS.Get(currentAcc).TdApi.GetGroupsInCommon(groupsInCommonUserId)
		if err != nil {
			log.Printf("failed to get groups in common: %d, `%s`, %s", groupsInCommonUserId, account.AS.Get(currentAcc).TdApi.GetChatName(groupsInCommonUserId), err)
		}
		for _, chatId := range chats.ChatIds {
			chat, err := account.AS.Get(currentAcc).TdApi.GetChat(chatId, true)
			var chatInfo ChatInfo
			if err != nil {
				chatInfo = ChatInfo{ChatId: chatId, ChatName: account.AS.Get(currentAcc).TdApi.GetChatName(chatId), Username: "ERROR " + err.Error()}
			} else {
				chatInfo = buildChatInfoByLocalChat(req.Context(), chat)
			}

			res.Chats = append(res.Chats, chatInfo)
		}
	} else {
		chatList := account.AS.Get(currentAcc).TdApi.GetStorage().GetSavedChats(folder)
		for _, chatPos := range chatList {
			chat, err := account.AS.Get(currentAcc).TdApi.GetChat(chatPos.ChatId, true)
			var chatInfo ChatInfo
			if err != nil {
				chatInfo = ChatInfo{ChatId: chatPos.ChatId, ChatName: account.AS.Get(currentAcc).TdApi.GetChatName(chatPos.ChatId), Username: "ERROR " + err.Error()}
			} else {
				chatInfo = buildChatInfoByLocalChat(req.Context(), chat)
			}

			res.Chats = append(res.Chats, chatInfo)
		}
	}

	renderTemplates(req, w, res, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/overview_table.gohtml`, `templates/chatlist.gohtml`)
}

func (wc *webController) processTgDelete(w http.ResponseWriter, req *http.Request) {
	chatId, err := strconv.ParseInt(req.PathValue("chat_id"), 10, 64)
	if chatId == 0 || err != nil {
		errorResponse(WebError{T: "Not found", Error: req.URL.Path}, 404, req, w)
		return
	}

	pattern := req.FormValue("pattern")
	if pattern == "" || len(pattern) < 3 {
		errorResponse(WebError{T: "Invalid pattern", Error: pattern}, 503, req, w)

		return
	}
	limit := 50
	if req.FormValue("limit") != "" {
		limit64, _ := strconv.ParseInt(req.FormValue("limit"), 10, 0)
		limit = int(limit64)
	}
	currentAcc := req.Context().Value("current_acc").(int64)

	var messageIds []int64
	messageIds = make([]int64, 0)
	var lastId int64 = 0
	for len(messageIds) < limit {
		history, err := account.AS.Get(currentAcc).TdApi.GetChatHistory(chatId, lastId)
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
	ok, err := account.AS.Get(currentAcc).TdApi.DeleteMessages(chatId, messageIds)
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

func (wc *webController) processAddAccount(w http.ResponseWriter, req *http.Request) {

	if tdAccount.CurrentAuthorizingAcc == nil && req.Method == "GET" {
		wc.st = newAccountState{}
	}
	if tdAccount.AuthorizerState == nil {
		wc.st.State = "start"
	} else if tdAccount.AuthorizerState.AuthorizationStateType() == client.TypeAuthorizationStateWaitCode {
		wc.st.State = "code"
		wc.st.Phone = tdAccount.CurrentAuthorizingAcc.Phone
	} else if tdAccount.AuthorizerState.AuthorizationStateType() == client.TypeAuthorizationStateWaitPassword {
		wc.st.State = "password"
		wc.st.Phone = tdAccount.CurrentAuthorizingAcc.Phone
	} else {
		wc.st.State = tdAccount.AuthorizerState.AuthorizationStateType()
		wc.st.Phone = tdAccount.CurrentAuthorizingAcc.Phone
	}

	if req.Method == "POST" {
		if tdAccount.CurrentAuthorizingAcc == nil {
			if req.FormValue("phone") != "" {
				tdAccount.CreateAccount(req.FormValue("phone"))
				if tdAccount.CurrentAuthorizingAcc.Status == consts.AccStatusActive {
					wc.st.State = "already_authorized"
					tdAccount.CurrentAuthorizingAcc = nil
				} else {
					wc.st.State = "wait"
				}
				wc.st.Phone = req.FormValue("phone")
			} else {
				wc.st.State = "wtf no phone?"
			}
		} else {
			if req.FormValue("code") != "" && wc.st.State == "code" {
				tdAccount.AuthParams <- req.FormValue("code")

				wc.st.State = "wait"
				wc.st.Phone = tdAccount.CurrentAuthorizingAcc.Phone
				wc.st.Code = req.FormValue("code")
			} else if req.FormValue("password") != "" && wc.st.State == "password" {
				tdAccount.AuthParams <- req.FormValue("password")

				wc.st.State = "wait"
				wc.st.Phone = tdAccount.CurrentAuthorizingAcc.Phone
				wc.st.Code = req.FormValue("code")
				wc.st.Password = req.FormValue("password")
			} else {
				wc.st.State = "must refresh form without POST"
			}
		}
		http.Redirect(w, req, "/new", 302)
		return
	} else {

		renderTemplates(req, w, wc.st, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/account_add.gohtml`)
	}
}

func (wc *webController) processTgLink(w http.ResponseWriter, req *http.Request) {
	var link string
	if req.FormValue("link") != "" {
		link = req.FormValue("link")
	} else {
		errorResponse(WebError{T: "Bad request", Error: "Invalid link"}, 400, req, w)
		return
	}

	linkInfo, LinkData, err := account.AS.Get(req.Context().Value("current_acc").(int64)).TdApi.GetLinkInfo(link)
	if err != nil {
		errorResponse(WebError{T: "Bad request", Error: err.Error()}, 400, req, w)
		return
	}
	respStruct := struct {
		T           string
		SourceLink  string
		LinkInfoRaw string
		LinkDataRaw string
	}{T: "Link info", SourceLink: link, LinkInfoRaw: string(helpers.JsonMarshalPretty(linkInfo)), LinkDataRaw: string(helpers.JsonMarshalPretty(LinkData))}

	renderTemplates(req, w, respStruct, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/link_info.gohtml`)
}

func (wc *webController) processFile(res http.ResponseWriter, req *http.Request) {
	if tryFile(req, res) {
		return
	}

	fileId := req.PathValue("file_id")
	file, err := account.AS.Get(req.Context().Value("current_acc").(int64)).TdApi.DownloadFileByRemoteId(fileId)

	if err != nil {
		errorResponse(WebError{T: "Attachment error", Error: err.Error()}, 502, req, res)

		return
	}
	if req.Context().Value("verbose").(bool) {
		renderTemplates(req, res, file)

		return
	}
	if file.Local.Path != "" {
		res.Header().Add("X-Local-path", base64.StdEncoding.EncodeToString([]byte(file.Local.Path)))
		http.ServeFile(res, req, file.Local.Path)

		return
	}

	errorResponse(WebError{T: "Invalid file", Error: file.Extra}, 504, req, res)

	return

}

func (wc *webController) catchAll(w http.ResponseWriter, r *http.Request) {
	if tryFile(r, w) {
		return
	}
	errorResponse(WebError{T: "Not found", Error: r.URL.Path}, 404, r, w)
}
