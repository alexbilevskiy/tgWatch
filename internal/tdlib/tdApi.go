package tdlib

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/alexbilevskiy/tgWatch/internal/config"
	"github.com/alexbilevskiy/tgWatch/internal/consts"
	"github.com/alexbilevskiy/tgWatch/internal/db"
	"github.com/alexbilevskiy/tgWatch/internal/helpers"
	"github.com/zelenin/go-tdlib/client"
)

type TdApi struct {
	log          *slog.Logger
	m            sync.RWMutex
	cfg          *config.Config
	dbData       *db.DbAccountData
	localChats   map[int64]*client.Chat
	chatFolders  []db.ChatFilter
	tdlibClient  *client.Client
	db           TdStorageInterface
	sentMessages sync.Map
}

type TdStorageInterface interface {
	DeleteChatFolder(ctx context.Context, folderId int32) (int64, error)
	ClearChatFilters(ctx context.Context)
	LoadChatFolders(ctx context.Context) []db.ChatFilter

	SaveChatFolder(ctx context.Context, chatFolder *client.ChatFolder, folderInfo *client.ChatFolderInfo)
	SaveAllChatPositions(ctx context.Context, chatId int64, positions []*client.ChatPosition)
	SaveChatPosition(ctx context.Context, chatId int64, chatPosition *client.ChatPosition)

	GetSavedChats(ctx context.Context, listId int32) []db.ChatPosition
}

func NewTdApi(logger *slog.Logger, cfg *config.Config, dbData *db.DbAccountData, tdMongo TdStorageInterface) *TdApi {
	return &TdApi{
		log:          logger,
		cfg:          cfg,
		db:           tdMongo,
		dbData:       dbData,
		localChats:   make(map[int64]*client.Chat),
		m:            sync.RWMutex{},
		sentMessages: sync.Map{},
	}
}

func (t *TdApi) RunTdlib(ctx context.Context) (*client.User, error) {
	t.chatFolders = t.db.LoadChatFolders(ctx)

	tdlibParameters := createTdlibParameters(t.cfg, t.dbData.DataDir)
	authorizer := NewClientAuthorizer(t.log, tdlibParameters)
	authParams := make(chan string)
	go authorizer.ChanInteractor(t.dbData.Phone, authParams)

	_, _ = client.SetLogVerbosityLevel(&client.SetLogVerbosityLevelRequest{
		NewVerbosityLevel: 1,
	})
	client.WithFallbackTimeout(60)

	tdlibClient, err := client.NewClient(authorizer, client.WithResultHandler(client.NewCallbackResultHandler(func(result client.Type) {
		t.UpdatesCallback(ctx, result)
	})))
	if err != nil {
		return nil, fmt.Errorf("create tdlib client: %w", err)
	}

	optionValue, err := tdlibClient.GetOption(&client.GetOptionRequest{Name: "version"})
	if err != nil {
		return nil, fmt.Errorf("create tdlib client: get version: %w", err)
	}

	t.log.Info("TDLib", "version", optionValue.(*client.OptionValueString).Value)

	me, err := tdlibClient.GetMe(ctx)
	if err != nil {
		return nil, fmt.Errorf("create tdlib client: get me: %w", err)
	}

	t.log.Info("Me", "phone", t.dbData.Phone, "fname", me.FirstName, "lname", me.LastName, "username", GetUsername(me.Usernames))

	//@NOTE: https://github.com/tdlib/td/issues/1005#issuecomment-613839507
	go func() {
		//for true {
		{
			req := &client.SetOptionRequest{Name: "online", Value: &client.OptionValueBoolean{Value: true}}
			ok, err := tdlibClient.SetOption(ctx, req)
			if err != nil {
				t.log.Warn("set online status", "error", err)
			} else {
				t.log.Info("set online status", "resp", helpers.JsonMarshalStr(ok))
			}
			//time.Sleep(10 * time.Second)
		}
	}()

	//req := &client.SetOptionRequest{Name: "ignore_background_updates", Value: &client.OptionValueBoolean{Value: false}}
	//ok, err := tdlibClient[acc].SetOption(req)
	//if err != nil {
	//	log.Printf("failed to set ignore_background_updates option: %s", err)
	//} else {
	//	log.Printf("Set ignore_background_updates option: %s", JsonMarshalStr(ok))
	//}
	t.tdlibClient = tdlibClient

	return me, nil
}

func (t *TdApi) GetChat(ctx context.Context, chatId int64, force bool) (*client.Chat, error) {
	t.m.RLock()
	fullChat, ok := t.localChats[chatId]
	t.m.RUnlock()
	if !force && ok {

		return fullChat, nil
	}
	req := &client.GetChatRequest{ChatId: chatId}
	fullChat, err := t.tdlibClient.GetChat(ctx, req)
	if err == nil {
		//fmt.Printf("Caching local chat %d\n", chatId))
		t.cacheChat(fullChat)
	}

	return fullChat, err
}

func (t *TdApi) cacheChat(chat *client.Chat) {
	t.m.Lock()
	t.localChats[chat.Id] = chat
	t.m.Unlock()
}

func (t *TdApi) GetUser(ctx context.Context, userId int64) (*client.User, error) {
	userReq := &client.GetUserRequest{UserId: userId}

	return t.tdlibClient.GetUser(ctx, userReq)
}

func (t *TdApi) GetSuperGroup(ctx context.Context, sgId int64) (*client.Supergroup, error) {
	sgReq := &client.GetSupergroupRequest{SupergroupId: sgId}

	return t.tdlibClient.GetSupergroup(ctx, sgReq)
}

func (t *TdApi) GetBasicGroup(ctx context.Context, groupId int64) (*client.BasicGroup, error) {
	bgReq := &client.GetBasicGroupRequest{BasicGroupId: groupId}

	return t.tdlibClient.GetBasicGroup(ctx, bgReq)
}

func (t *TdApi) GetGroupsInCommon(ctx context.Context, userId int64) (*client.Chats, error) {
	cgReq := &client.GetGroupsInCommonRequest{UserId: userId, Limit: 500}

	return t.tdlibClient.GetGroupsInCommon(ctx, cgReq)
}

func (t *TdApi) DownloadFile(ctx context.Context, id int32) (*client.File, error) {
	req := client.DownloadFileRequest{FileId: id, Priority: 1, Synchronous: true}
	file, err := t.tdlibClient.DownloadFile(ctx, &req)
	if err != nil {
		return nil, fmt.Errorf("download file by id: %w", err)
	}

	return file, nil
}

func (t *TdApi) DownloadFileByRemoteId(ctx context.Context, id string) (*client.File, error) {
	remoteFileReq := client.GetRemoteFileRequest{RemoteFileId: id}
	remoteFile, err := t.tdlibClient.GetRemoteFile(ctx, &remoteFileReq)
	if err != nil {

		return nil, fmt.Errorf("download file by remote id: %w", err)
	}

	return t.DownloadFile(ctx, remoteFile.Id)
}

func (t *TdApi) GetCustomEmoji(ctx context.Context, customEmojisIds []int64) (*client.Stickers, error) {
	customEmojisIdsJson := make([]client.JsonInt64, 0)
	for _, id := range customEmojisIds {
		customEmojisIdsJson = append(customEmojisIdsJson, client.JsonInt64(id))
	}
	customEmojisReq := client.GetCustomEmojiStickersRequest{CustomEmojiIds: customEmojisIdsJson}
	customEmojis, err := t.tdlibClient.GetCustomEmojiStickers(ctx, &customEmojisReq)
	if err != nil {
		return nil, fmt.Errorf("get custom emoji by ids: %w", err)
	}

	return customEmojis, nil
}

func (t *TdApi) markAsRead(ctx context.Context, chatId int64, messageId int64) error {
	req := &client.ViewMessagesRequest{ChatId: chatId, MessageIds: append(make([]int64, 0), messageId), ForceRead: true}
	_, err := t.tdlibClient.ViewMessages(ctx, req)

	return err
}

func (t *TdApi) GetLink(ctx context.Context, chatId int64, messageId int64) string {
	chat, err := t.GetChat(ctx, chatId, false)
	if err != nil {
		t.log.Warn("GetLink: chat not found", "phone", t.dbData.Phone, "chat", chatId, "err", err)
		return ""
	}
	if chat.Type.ChatTypeConstructor() != client.ConstructorChatTypeSupergroup {
		//fmt.Printf("GetLink: not available for chat `%s` (%d) with type %s", chat.Title, chatId, chat.Type.ChatTypeType()))
		return ""
	}

	linkReq := &client.GetMessageLinkRequest{ChatId: chatId, MessageId: messageId}
	link, err := t.tdlibClient.GetMessageLink(ctx, linkReq)
	if err != nil {
		if err.Error() != "400 Message not found" {
			t.log.Warn("get msg link by chat id", "chat", chatId, "msg", messageId, "error", err)
		}

		return ""
	}

	return link.Link
}

func (t *TdApi) AddChatsToFolder(ctx context.Context, chats []int64, folder int32) error {
	for _, chatId := range chats {
		_, err := t.GetChat(ctx, chatId, true)
		if err != nil {
			t.log.Warn("get chat before adding to folder", "phone", t.dbData.Phone, "chat", chatId, "error", err)
			continue
		}

		chatList := &client.ChatListFolder{ChatFolderId: folder}
		req := client.AddChatToListRequest{ChatId: chatId, ChatList: chatList}
		_, err = t.tdlibClient.AddChatToList(ctx, &req)
		if err != nil {
			t.log.Warn("add chat to list", "chat", chatId, "listid", folder, "error", err)
		} else {
			t.log.Info("added chat to list", "chat", chatId, "listid", folder)
		}
	}

	return nil
}

func (t *TdApi) SendMessage(ctx context.Context, text string, chatId int64, replyToMessageId *int64) {
	mtext := &client.FormattedText{Text: text}
	content := &client.InputMessageText{Text: mtext}
	var req *client.SendMessageRequest
	if replyToMessageId == nil {
		req = &client.SendMessageRequest{ChatId: chatId, InputMessageContent: content}
	} else {
		replyTo := client.InputMessageReplyToMessage{MessageId: *replyToMessageId}
		req = &client.SendMessageRequest{ChatId: chatId, ReplyTo: &replyTo, InputMessageContent: content}
	}
	message, err := t.tdlibClient.SendMessage(ctx, req)
	//@TODO: use t.sentMessages etc
	if err != nil {
		t.log.Warn("send message", "chat", chatId, "error", err)
	} else {
		t.log.Info("sent message", "chat", chatId, "virtual_id", message.Id)
	}
}

func (t *TdApi) GetLinkInfo(ctx context.Context, link string) (client.InternalLinkType, interface{}, error) {
	linkTypeReq := &client.GetInternalLinkTypeRequest{Link: link}
	linkType, err := t.tdlibClient.GetInternalLinkType(ctx, linkTypeReq)
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("get link type error: %s", err.Error()))
	}
	switch linkType.InternalLinkTypeConstructor() {
	case client.ConstructorInternalLinkTypeMessage:
		linkType := linkType.(*client.InternalLinkTypeMessage)
		messageLinkInfoReq := &client.GetMessageLinkInfoRequest{Url: link}
		messageLinkInfo, err := t.tdlibClient.GetMessageLinkInfo(ctx, messageLinkInfoReq)
		if err == nil {
			return linkType, messageLinkInfo.Message, nil
		}

		return linkType, err, nil

	case client.ConstructorInternalLinkTypePublicChat:
		linkType := linkType.(*client.InternalLinkTypePublicChat)
		publicChatReq := &client.SearchPublicChatRequest{Username: linkType.ChatUsername}
		publicChat, err := t.tdlibClient.SearchPublicChat(ctx, publicChatReq)
		if err == nil {
			return linkType, publicChat, nil
		}

		return linkType, err, nil

	case client.ConstructorInternalLinkTypeChatInvite:
		linkType := linkType.(*client.InternalLinkTypeChatInvite)
		chatInviteLinkReq := &client.CheckChatInviteLinkRequest{InviteLink: linkType.InviteLink}
		chatInviteLink, err := t.tdlibClient.CheckChatInviteLink(ctx, chatInviteLinkReq)
		if err == nil {
			return linkType, chatInviteLink, nil
		}

		return linkType, err, nil

	default:
		return linkType, errors.New(fmt.Sprintf("unknown link type: %s", linkType.InternalLinkTypeConstructor())), nil
	}
}

func (t *TdApi) GetMessage(ctx context.Context, chatId int64, messageId int64) (*client.Message, error) {
	t.log.Info("get message", "chat_id", chatId, "message_id", messageId)
	var err error

	openChatReq := &client.OpenChatRequest{ChatId: chatId}
	_, err = t.tdlibClient.OpenChat(ctx, openChatReq)
	if err != nil {

		return nil, errors.New(fmt.Sprintf("open chat error: %s", err.Error()))
	}

	messageIds := make([]int64, 0)
	messageIds = append(messageIds, messageId)
	viewMessagesReq := &client.ViewMessagesRequest{ChatId: chatId, MessageIds: messageIds}
	_, err = t.tdlibClient.ViewMessages(ctx, viewMessagesReq)
	if err != nil {

		return nil, errors.New(fmt.Sprintf("failed to view message: %s", err.Error()))
	}
	//log.Printf("sleeping before get message %d/%d", chatId, messageId)
	//time.Sleep(time.Second * 5)

	getMessageReq := &client.GetMessageRequest{ChatId: chatId, MessageId: messageId}
	message, err := t.tdlibClient.GetMessage(ctx, getMessageReq)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("get message error: %s", err.Error()))
	}

	return message, nil
}

func (t *TdApi) getChatFolder(ctx context.Context, folderId int32) (*client.ChatFolder, error) {
	req := &client.GetChatFolderRequest{ChatFolderId: folderId}
	return t.tdlibClient.GetChatFolder(ctx, req)
}

func (t *TdApi) LoadChatHistory(ctx context.Context, chatId int64, fromMessageId int64, offset int32) (*client.Messages, error) {
	chatHistoryRequest := client.GetChatHistoryRequest{ChatId: chatId, Offset: offset, FromMessageId: fromMessageId, OnlyLocal: false, Limit: 50}
	messages, err := t.tdlibClient.GetChatHistory(ctx, &chatHistoryRequest)
	if err != nil {

		return nil, errors.New(fmt.Sprintf("failed to load history: %s", err.Error()))
	}

	return messages, nil
}

func (t *TdApi) MarkJoinAsRead(ctx context.Context, chatId int64, messageId int64) {
	chat, err := t.GetChat(ctx, chatId, true)
	if err != nil {
		t.log.Warn("cannot update unread count because chat not found", "phone", t.dbData.Phone, "chat", chatId, "error", err)

		return
	}
	//name := GetChatName(acc, chatId)

	if chat.UnreadCount != 1 {
		//fmt.Printf("Chat `%s` %d unread count: %d>1, not marking as read\n", name, chatId, chat.UnreadCount))
		return
	}
	//fmt.Printf("Chat `%s` %d unread count: %d, marking join as read\n", name, chatId, chat.UnreadCount))

	err = t.markAsRead(ctx, chatId, messageId)
	if err != nil {
		fmt.Printf("Cannot mark as read chat %d, message %d: %s\n", chatId, messageId, err.Error())

		return
	}
	chat, err = t.GetChat(ctx, chatId, true)
	if err != nil {
		fmt.Printf("Cannot get NEW unread count because chat %d not found: %s\n", chatId, err.Error())

		return
	}
	//fmt.Printf("NEW Chat `%s` %d unread count: %d\n", name, chatId, chat.UnreadCount))
}

func (t *TdApi) GetSenderName(ctx context.Context, sender client.MessageSender) string {
	chat, err := t.GetSenderObj(ctx, sender)
	if err != nil {

		return err.Error()
	}
	if sender.MessageSenderConstructor() == client.ConstructorMessageSenderChat {
		name := fmt.Sprintf("%s", chat.(*client.Chat).Title)
		if name == "" {
			name = fmt.Sprintf("no_name %d", chat.(*client.Chat).Id)
		}
		return name
	} else if sender.MessageSenderConstructor() == client.ConstructorMessageSenderUser {
		user := chat.(*client.User)
		return GetUserFullname(user)
	}

	return "unkown_chattype"
}

func (t *TdApi) GetSenderObj(ctx context.Context, sender client.MessageSender) (interface{}, error) {
	if sender.MessageSenderConstructor() == client.ConstructorMessageSenderChat {
		chatId := sender.(*client.MessageSenderChat).ChatId
		chat, err := t.GetChat(ctx, chatId, false)
		if err != nil {
			t.log.Warn("request sender chat info by id", "phone", t.dbData.Phone, "chat", chatId, "error", err)

			return nil, errors.New("unknown chat")
		}

		return chat, nil
	} else if sender.MessageSenderConstructor() == client.ConstructorMessageSenderUser {
		userId := sender.(*client.MessageSenderUser).UserId
		user, err := t.GetUser(ctx, userId)
		if err != nil {
			t.log.Warn("request sender user info by id", "phone", t.dbData.Phone, "user", userId, "error", err)

			return nil, errors.New("unknown user")
		}

		return user, nil
	}

	return nil, errors.New("unknown sender type")
}

func (t *TdApi) GetChatName(ctx context.Context, chatId int64) string {
	fullChat, err := t.GetChat(ctx, chatId, false)
	if err != nil {
		t.log.Warn("get chat for name", "phone", t.dbData.Phone, "chat", chatId, "error", err)

		return "no_title"
	}
	name := fmt.Sprintf("%s", fullChat.Title)
	if name == "" {
		name = fmt.Sprintf("no_name %d", chatId)
	}

	return name
}

func (t *TdApi) GetChatUsername(ctx context.Context, chatId int64) string {
	chat, err := t.GetChat(ctx, chatId, false)
	if err != nil {
		t.log.Warn("get chat for username", "phone", t.dbData.Phone, "chat", chatId, "error", err)

		return ""
	}
	switch chat.Type.ChatTypeConstructor() {
	case client.ConstructorChatTypeSupergroup:
		typ := chat.Type.(*client.ChatTypeSupergroup)
		sg, err := t.GetSuperGroup(ctx, typ.SupergroupId)
		if err != nil {
			t.log.Warn("GetSuperGroup for username", "phone", t.dbData.Phone, "chat", typ.SupergroupId, "error", err)
			return ""
		}
		return GetUsername(sg.Usernames)
	case client.ConstructorChatTypePrivate:
		typ := chat.Type.(*client.ChatTypePrivate)
		user, err := t.GetUser(ctx, typ.UserId)
		if err != nil {
			t.log.Warn("GetUser for username", "phone", t.dbData.Phone, "user", typ.UserId, "error", err)
			return ""
		}
		return GetUsername(user.Usernames)
	}

	return ""
}

func (t *TdApi) LoadChatsList(ctx context.Context, listId int32) {
	var chatList client.ChatList
	d, err := t.db.DeleteChatFolder(ctx, listId)
	if err != nil {
		t.log.Warn("delete chats by list", "list_id", listId, "error", err)
	} else {
		t.log.Warn("deleted chats by listid", "count", d, "list_id", listId)
	}

	switch listId {
	case consts.ClMain:
		chatList = &client.ChatListMain{}
		t.log.Info("requesting LoadChats for main list", "type", chatList.ChatListConstructor())
	case consts.ClArchive:
		chatList = &client.ChatListArchive{}
		t.log.Info("requesting LoadChats for archive", "type", chatList.ChatListConstructor())
	default:
		chatList = &client.ChatListFolder{ChatFolderId: listId}
		t.log.Info("requesting LoadChats for folder", "id", chatList.(*client.ChatListFolder).ChatFolderId)
	}

	loadChatsReq := &client.LoadChatsRequest{ChatList: chatList, Limit: 500}
	_, err = t.tdlibClient.LoadChats(ctx, loadChatsReq)

	if err == nil {
		t.log.Info("load chats ok, chat list updated will be received asynchronous")
		//everything ok
		return
	}
	if err.Error() != "404 Not Found" {
		t.log.Error("LoadChats", "phone", t.dbData.Phone, "error", err)
		//dunno what to do yet
		panic("LoadChats")
	}
	//@see https://github.com/tdlib/td/blob/fb39e5d74667db915a75a5e58065c59af8e7d8d6/td/generate/scheme/td_api.tl#L4171

	t.log.Info("all chats already loaded, trying to get them")
	getChatsReq := &client.GetChatsRequest{ChatList: chatList, Limit: 500}
	chats, err := t.tdlibClient.GetChats(ctx, getChatsReq)
	if err != nil {
		t.log.Error("GetChats", "phone", t.dbData.Phone, "error", err)
	}
	for _, chat := range chats.ChatIds {
		t.db.SaveChatPosition(ctx, chat, &client.ChatPosition{
			List:     chatList,
			Order:    0,
			IsPinned: false,
			Source:   nil,
		})
	}
}

func (t *TdApi) SaveChatFilters(ctx context.Context, chatFoldersUpdate *client.UpdateChatFolders) {
	t.log.Info("chat filters update", "type", chatFoldersUpdate.GetConstructor())
	//@TODO: why was commented?
	//t.tdMongo.ClearChatFilters()
	var wg sync.WaitGroup

	for _, folderInfo := range chatFoldersUpdate.ChatFolders {
		existed := false
		for _, existningFilter := range t.chatFolders {
			if existningFilter.Id == folderInfo.Id {
				existed = true
				break
			}
		}
		if existed {
			//log.Printf("Existing chat folder: id: %d, n: %s", folderInfo.Id, folderInfo.Title)
			continue
		}
		t.log.Info("new chat folder", "id", folderInfo.Id, "name", folderInfo.Name.Text.Text)

		wg.Add(1)
		go func(folderInfo *client.ChatFolderInfo, wg *sync.WaitGroup) {
			defer wg.Done()
			chatFolder, err := t.getChatFolder(ctx, folderInfo.Id)
			if err != nil {
				t.log.Warn("load chat folder", "id", folderInfo.Id, "name", folderInfo.Name.Text.Text, "error", err)

				return
			}
			t.db.SaveChatFolder(ctx, chatFolder, folderInfo)
			t.log.Info("chat folder LOADED", "id", folderInfo.Id, "name", folderInfo.Name.Text.Text)
		}(folderInfo, &wg)
		//time.Sleep(time.Second * 2)
	}
	wg.Wait()

	for _, existningFolder := range t.chatFolders {
		deleted := true
		for _, folderInfo := range chatFoldersUpdate.ChatFolders {
			if folderInfo.Id == existningFolder.Id {
				deleted = false
				continue
			}
		}
		if !deleted {
			continue
		}
		t.log.Info("deleted chat folder", "id", existningFolder.Id, "name", existningFolder.Title)
	}

	t.chatFolders = t.db.LoadChatFolders(ctx)
}

func (t *TdApi) SaveChatAddedToList(ctx context.Context, upd *client.UpdateChatAddedToList) {
	//j, _ := json.Marshal(upd)
	//log.Printf("saving chat added to list %s : %s", string(j))
	switch upd.ChatList.ChatListConstructor() {
	case client.ConstructorChatListMain:
		position := client.ChatPosition{
			List:     &client.ChatListMain{},
			Order:    0,
			IsPinned: false,
			Source:   nil,
		}
		t.db.SaveChatPosition(ctx, upd.ChatId, &position)
	case client.ConstructorChatListArchive:
		position := client.ChatPosition{
			List:     &client.ChatListArchive{},
			Order:    0,
			IsPinned: false,
			Source:   nil,
		}
		t.db.SaveChatPosition(ctx, upd.ChatId, &position)
	case client.ConstructorChatListFolder:
		position := client.ChatPosition{
			List:     &client.ChatListFolder{ChatFolderId: upd.ChatList.(*client.ChatListFolder).ChatFolderId},
			Order:    0,
			IsPinned: false,
			Source:   nil,
		}
		t.db.SaveChatPosition(ctx, upd.ChatId, &position)
	default:
		t.log.Warn("unknown chatlist type", "type", upd.ChatList.ChatListConstructor())
	}
}

func (t *TdApi) RemoveChatRemovedFromList(ctx context.Context, upd *client.UpdateChatRemovedFromList) {
	j, _ := json.Marshal(upd)
	t.log.Warn("NOT IMPLEMENTED: removing chat removed from list", "phone", t.dbData.Phone, "upd", string(j))
	return
	switch upd.ChatList.ChatListConstructor() {
	case client.ConstructorChatListMain:
		position := client.ChatPosition{
			List:     &client.ChatListMain{},
			Order:    0,
			IsPinned: false,
			Source:   nil,
		}
		t.db.SaveChatPosition(ctx, upd.ChatId, &position)
	case client.ConstructorChatListArchive:
		position := client.ChatPosition{
			List:     &client.ChatListArchive{},
			Order:    0,
			IsPinned: false,
			Source:   nil,
		}
		t.db.SaveChatPosition(ctx, upd.ChatId, &position)
	case client.ConstructorChatListFolder:
		position := client.ChatPosition{
			List:     &client.ChatListFolder{ChatFolderId: upd.ChatList.(*client.ChatListFolder).ChatFolderId},
			Order:    0,
			IsPinned: false,
			Source:   nil,
		}
		t.db.SaveChatPosition(ctx, upd.ChatId, &position)
	default:
		t.log.Warn("unknown chatlist type", "type", upd.ChatList.ChatListConstructor())
	}
}

func (t *TdApi) GetChatFolders() []db.ChatFilter {
	//@TODO: mutex?
	return t.chatFolders
}

func (t *TdApi) GetLocalChats() map[int64]*client.Chat {
	//@TODO: mutex?
	return t.localChats
}

func (t *TdApi) GetTdlibOption(optionName string) (client.OptionValue, error) {
	req := client.GetOptionRequest{Name: optionName}

	return t.tdlibClient.GetOption(&req)
}

func (t *TdApi) GetActiveSessions(ctx context.Context) (*client.Sessions, error) {

	return t.tdlibClient.GetActiveSessions(ctx)
}

func (t *TdApi) GetChatHistory(ctx context.Context, chatId int64, lastId int64) (*client.Messages, error) {
	req := &client.GetChatHistoryRequest{ChatId: chatId, Limit: 100, FromMessageId: lastId, Offset: 0}

	return t.tdlibClient.GetChatHistory(ctx, req)
}

func (t *TdApi) DeleteMessages(ctx context.Context, chatId int64, messageIds []int64) (*client.Ok, error) {
	req := &client.DeleteMessagesRequest{ChatId: chatId, MessageIds: messageIds}

	return t.tdlibClient.DeleteMessages(ctx, req)
}

func (t *TdApi) GetChatMember(ctx context.Context, chatId int64) (*client.ChatMember, error) {
	m := client.MessageSenderUser{UserId: t.dbData.Id}
	req := &client.GetChatMemberRequest{ChatId: chatId, MemberId: &m}

	return t.tdlibClient.GetChatMember(ctx, req)
}

func (t *TdApi) GetScheduledMessages(ctx context.Context, chatId int64) (*client.Messages, error) {
	req := &client.GetChatScheduledMessagesRequest{ChatId: chatId}

	return t.tdlibClient.GetChatScheduledMessages(ctx, req)
}

func (t *TdApi) ScheduleForwardedMessage(ctx context.Context, targetChatId int64, fromChatId int64, messageIds []int64, sendAtDate int32, sendCopy bool) (*client.Messages, error) {
	opts := &client.MessageSendOptions{SchedulingState: &client.MessageSchedulingStateSendAtDate{SendDate: sendAtDate}}
	req := &client.ForwardMessagesRequest{ChatId: targetChatId, MessageIds: messageIds, FromChatId: fromChatId, Options: opts, SendCopy: sendCopy}

	res, err := t.tdlibClient.ForwardMessages(ctx, req)
	if err != nil {
		return res, err
	}
	actualMessages := make([]*client.Message, 0)

	now := time.Now()
	for {
		for _, m := range res.Messages {
			if sent, ok := t.sentMessages.Load(m.Id); ok {
				actualMessages = append(actualMessages, sent.(*client.Message))
				t.sentMessages.Delete(m.Id)
			}
		}
		if len(actualMessages) == len(res.Messages) {
			break
		}
		if time.Since(now) > 5*time.Second {

			return nil, errors.New("timeout while waiting for actual send")
		}
		time.Sleep(500 * time.Millisecond)
	}
	res.Messages = actualMessages

	return res, err
}

func (t *TdApi) EditMessageSchedulingState(ctx context.Context, chatId int64, messageId int64, schedulingStateType string, sendDate int32) (*client.Ok, error) {
	var schedulingState client.MessageSchedulingState
	switch schedulingStateType {
	case client.ConstructorMessageSchedulingStateSendAtDate:
		schedulingState = &client.MessageSchedulingStateSendAtDate{SendDate: sendDate}
	case client.ConstructorMessageSchedulingStateSendWhenOnline:
		schedulingState = &client.MessageSchedulingStateSendWhenOnline{}
	}

	req := &client.EditMessageSchedulingStateRequest{
		ChatId:          chatId,
		MessageId:       messageId,
		SchedulingState: schedulingState,
	}

	return t.tdlibClient.EditMessageSchedulingState(ctx, req)
}

func (t *TdApi) GetStorage() TdStorageInterface {
	//@TODO: mutex?
	return t.db
}

func (t *TdApi) Close(ctx context.Context) {

	t.tdlibClient.Close(ctx)
}

func createTdlibParameters(cfg *config.Config, dataDir string) *client.SetTdlibParametersRequest {
	return &client.SetTdlibParametersRequest{
		UseTestDc:           false,
		DatabaseDirectory:   filepath.Join(cfg.TDataDir, dataDir, "database"),
		FilesDirectory:      filepath.Join(cfg.TDataDir, dataDir, "files"),
		UseFileDatabase:     true,
		UseChatInfoDatabase: true,
		UseMessageDatabase:  true,
		UseSecretChats:      false,
		ApiId:               cfg.ApiId,
		ApiHash:             cfg.ApiHash,
		SystemLanguageCode:  "en",
		DeviceModel:         "Linux",
		SystemVersion:       "1.0.0",
		ApplicationVersion:  "1.0.0",
	}
}
