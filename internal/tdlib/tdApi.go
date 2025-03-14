package tdlib

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
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
	DeleteChatFolder(folderId int32) (int64, error)
	ClearChatFilters()
	LoadChatFolders() []db.ChatFilter

	SaveChatFolder(chatFolder *client.ChatFolder, folderInfo *client.ChatFolderInfo)
	SaveAllChatPositions(chatId int64, positions []*client.ChatPosition)
	SaveChatPosition(chatId int64, chatPosition *client.ChatPosition)

	GetSavedChats(listId int32) []db.ChatPosition
}

func NewTdApi(cfg *config.Config, dbData *db.DbAccountData, tdMongo TdStorageInterface) *TdApi {
	t := &TdApi{
		cfg: cfg,
	}
	t.db = tdMongo
	t.dbData = dbData

	t.localChats = make(map[int64]*client.Chat)
	t.chatFolders = t.db.LoadChatFolders()
	t.m = sync.RWMutex{}
	t.sentMessages = sync.Map{}

	return t
}

func (t *TdApi) RunTdlib() *client.User {
	tdlibParameters := createTdlibParameters(t.cfg, t.dbData.DataDir)
	authorizer := ClientAuthorizer(tdlibParameters)
	authParams := make(chan string)
	go ChanInteractor(authorizer, t.dbData.Phone, authParams)

	logVerbosity := client.WithLogVerbosity(&client.SetLogVerbosityLevelRequest{
		NewVerbosityLevel: 1,
	})
	//client.WithCatchTimeout(60)

	tdlibClient, err := client.NewClient(authorizer, logVerbosity)
	if err != nil {
		log.Fatalf("NewClient error: %s", err)
	}

	optionValue, err := tdlibClient.GetOption(&client.GetOptionRequest{
		Name: "version",
	})
	if err != nil {
		log.Fatalf("GetOption error: %s", err)
	}

	log.Printf("TDLib version: %s", optionValue.(*client.OptionValueString).Value)

	me, err := tdlibClient.GetMe()
	if err != nil {
		log.Fatalf("GetMe error: %s", err)
	}

	log.Printf("Me: %s %s [%s]", me.FirstName, me.LastName, GetUsername(me.Usernames))

	//@NOTE: https://github.com/tdlib/td/issues/1005#issuecomment-613839507
	go func() {
		//for true {
		{
			req := &client.SetOptionRequest{Name: "online", Value: &client.OptionValueBoolean{Value: true}}
			ok, err := tdlibClient.SetOption(req)
			if err != nil {
				log.Printf("failed to set online option: %s", err)
			} else {
				log.Printf("Set online status: %s", helpers.JsonMarshalStr(ok))
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

	return me
}

func (t *TdApi) GetChat(chatId int64, force bool) (*client.Chat, error) {
	t.m.RLock()
	fullChat, ok := t.localChats[chatId]
	t.m.RUnlock()
	if !force && ok {

		return fullChat, nil
	}
	req := &client.GetChatRequest{ChatId: chatId}
	fullChat, err := t.tdlibClient.GetChat(req)
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

func (t *TdApi) GetUser(userId int64) (*client.User, error) {
	userReq := &client.GetUserRequest{UserId: userId}

	return t.tdlibClient.GetUser(userReq)
}

func (t *TdApi) GetSuperGroup(sgId int64) (*client.Supergroup, error) {
	sgReq := &client.GetSupergroupRequest{SupergroupId: sgId}

	return t.tdlibClient.GetSupergroup(sgReq)
}

func (t *TdApi) GetBasicGroup(groupId int64) (*client.BasicGroup, error) {
	bgReq := &client.GetBasicGroupRequest{BasicGroupId: groupId}

	return t.tdlibClient.GetBasicGroup(bgReq)
}

func (t *TdApi) GetGroupsInCommon(userId int64) (*client.Chats, error) {
	cgReq := &client.GetGroupsInCommonRequest{UserId: userId, Limit: 500}

	return t.tdlibClient.GetGroupsInCommon(cgReq)
}

func (t *TdApi) DownloadFile(id int32) (*client.File, error) {
	req := client.DownloadFileRequest{FileId: id, Priority: 1, Synchronous: true}
	file, err := t.tdlibClient.DownloadFile(&req)
	if err != nil {
		//log.Printf("Cannot download file: %s %d", err, id)

		return nil, errors.New("downloading error: " + err.Error())
	}

	return file, nil
}

func (t *TdApi) DownloadFileByRemoteId(id string) (*client.File, error) {
	remoteFileReq := client.GetRemoteFileRequest{RemoteFileId: id}
	remoteFile, err := t.tdlibClient.GetRemoteFile(&remoteFileReq)
	if err != nil {
		//log.Printf("cannot get remote file info: %s %s", err, id)

		return nil, errors.New("remoteFile request error: " + err.Error())
	}
	//if remoteFile.Local.IsDownloadingCompleted {
	//	log.Printf("Not dowloading file again: %s", remoteFile.Local.Path)
	//
	//	return remoteFile, nil
	//}

	return t.DownloadFile(remoteFile.Id)
}

func (t *TdApi) GetCustomEmoji(customEmojisIds []int64) (*client.Stickers, error) {
	customEmojisIdsJson := make([]client.JsonInt64, 0)
	for _, id := range customEmojisIds {
		customEmojisIdsJson = append(customEmojisIdsJson, client.JsonInt64(id))
	}
	customEmojisReq := client.GetCustomEmojiStickersRequest{CustomEmojiIds: customEmojisIdsJson}
	customEmojis, err := t.tdlibClient.GetCustomEmojiStickers(&customEmojisReq)
	if err != nil {
		return nil, errors.New("custom emoji error: " + err.Error())
	}

	return customEmojis, nil
}

func (t *TdApi) markAsRead(chatId int64, messageId int64) error {
	req := &client.ViewMessagesRequest{ChatId: chatId, MessageIds: append(make([]int64, 0), messageId), ForceRead: true}
	_, err := t.tdlibClient.ViewMessages(req)

	return err
}

func (t *TdApi) GetLink(chatId int64, messageId int64) string {
	chat, err := t.GetChat(chatId, false)
	if err != nil {
		log.Printf("GetLink: chat %d not found: %s", chatId, err.Error())
		return ""
	}
	if chat.Type.ChatTypeType() != client.TypeChatTypeSupergroup {
		//fmt.Printf("GetLink: not available for chat `%s` (%d) with type %s", chat.Title, chatId, chat.Type.ChatTypeType()))
		return ""
	}

	linkReq := &client.GetMessageLinkRequest{ChatId: chatId, MessageId: messageId}
	link, err := t.tdlibClient.GetMessageLink(linkReq)
	if err != nil {
		if err.Error() != "400 Message not found" {
			log.Printf("Failed to get msg link by chat id %d, msg id %d: %s", chatId, messageId, err)
		}

		return ""
	}

	return link.Link
}

func (t *TdApi) AddChatsToFolder(chats []int64, folder int32) error {
	for _, chatId := range chats {
		_, err := t.GetChat(chatId, true)
		if err != nil {
			log.Printf("failed to get chat before adding to folder: %d %s", chatId, err.Error())
			continue
		}

		chatList := &client.ChatListFolder{ChatFolderId: folder}
		req := client.AddChatToListRequest{ChatId: chatId, ChatList: chatList}
		_, err = t.tdlibClient.AddChatToList(&req)
		if err != nil {
			log.Printf("failed to add chat %d to list %d: %s", chatId, folder, err.Error())
		} else {
			log.Printf("added chat %d to list %d", chatId, folder)
		}
	}

	return nil
}

func (t *TdApi) SendMessage(text string, chatId int64, replyToMessageId *int64) {
	mtext := &client.FormattedText{Text: text}
	content := &client.InputMessageText{Text: mtext}
	var req *client.SendMessageRequest
	if replyToMessageId == nil {
		req = &client.SendMessageRequest{ChatId: chatId, InputMessageContent: content}
	} else {
		replyTo := client.InputMessageReplyToMessage{MessageId: *replyToMessageId}
		req = &client.SendMessageRequest{ChatId: chatId, ReplyTo: &replyTo, InputMessageContent: content}
	}
	message, err := t.tdlibClient.SendMessage(req)
	//@TODO: use t.sentMessages etc
	if err != nil {
		log.Printf("Failed to send message to chat %d: %s", chatId, err.Error())
	} else {
		log.Printf("Sent message to chat %d! new (virtual) message id: %d", chatId, message.Id)
	}
}

func (t *TdApi) GetLinkInfo(link string) (client.InternalLinkType, interface{}, error) {
	linkTypeReq := &client.GetInternalLinkTypeRequest{Link: link}
	linkType, err := t.tdlibClient.GetInternalLinkType(linkTypeReq)
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("get link type error: %s", err.Error()))
	}
	switch linkType.InternalLinkTypeType() {
	case client.TypeInternalLinkTypeMessage:
		linkType := linkType.(*client.InternalLinkTypeMessage)
		messageLinkInfoReq := &client.GetMessageLinkInfoRequest{Url: link}
		messageLinkInfo, err := t.tdlibClient.GetMessageLinkInfo(messageLinkInfoReq)
		if err == nil {
			return linkType, messageLinkInfo.Message, nil
		}

		return linkType, err, nil

	case client.TypeInternalLinkTypePublicChat:
		linkType := linkType.(*client.InternalLinkTypePublicChat)
		publicChatReq := &client.SearchPublicChatRequest{Username: linkType.ChatUsername}
		publicChat, err := t.tdlibClient.SearchPublicChat(publicChatReq)
		if err == nil {
			return linkType, publicChat, nil
		}

		return linkType, err, nil

	case client.TypeInternalLinkTypeChatInvite:
		linkType := linkType.(*client.InternalLinkTypeChatInvite)
		chatInviteLinkReq := &client.CheckChatInviteLinkRequest{InviteLink: linkType.InviteLink}
		chatInviteLink, err := t.tdlibClient.CheckChatInviteLink(chatInviteLinkReq)
		if err == nil {
			return linkType, chatInviteLink, nil
		}

		return linkType, err, nil

	default:
		return linkType, errors.New(fmt.Sprintf("unknown link type: %s", linkType.InternalLinkTypeType())), nil
	}
}

func (t *TdApi) GetMessage(chatId int64, messageId int64) (*client.Message, error) {
	log.Printf("get message %d/%d", chatId, messageId)
	var err error

	openChatReq := &client.OpenChatRequest{ChatId: chatId}
	_, err = t.tdlibClient.OpenChat(openChatReq)
	if err != nil {

		return nil, errors.New(fmt.Sprintf("open chat error: %s", err.Error()))
	}

	messageIds := make([]int64, 0)
	messageIds = append(messageIds, messageId)
	viewMessagesReq := &client.ViewMessagesRequest{ChatId: chatId, MessageIds: messageIds}
	_, err = t.tdlibClient.ViewMessages(viewMessagesReq)
	if err != nil {

		return nil, errors.New(fmt.Sprintf("failed to view message: %s", err.Error()))
	}
	//log.Printf("sleeping before get message %d/%d", chatId, messageId)
	//time.Sleep(time.Second * 5)

	getMessageReq := &client.GetMessageRequest{ChatId: chatId, MessageId: messageId}
	message, err := t.tdlibClient.GetMessage(getMessageReq)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("get message error: %s", err.Error()))
	}

	return message, nil
}

func (t *TdApi) getChatFolder(folderId int32) (*client.ChatFolder, error) {
	req := &client.GetChatFolderRequest{ChatFolderId: folderId}
	return t.tdlibClient.GetChatFolder(req)
}

func (t *TdApi) LoadChatHistory(chatId int64, fromMessageId int64, offset int32) (*client.Messages, error) {
	chatHistoryRequest := client.GetChatHistoryRequest{ChatId: chatId, Offset: offset, FromMessageId: fromMessageId, OnlyLocal: false, Limit: 50}
	messages, err := t.tdlibClient.GetChatHistory(&chatHistoryRequest)
	if err != nil {

		return nil, errors.New(fmt.Sprintf("failed to load history: %s", err.Error()))
	}

	return messages, nil
}

func (t *TdApi) MarkJoinAsRead(chatId int64, messageId int64) {
	chat, err := t.GetChat(chatId, true)
	if err != nil {
		fmt.Printf("Cannot update unread count because chat %d not found: %s\n", chatId, err.Error())

		return
	}
	//name := GetChatName(acc, chatId)

	if chat.UnreadCount != 1 {
		//fmt.Printf("Chat `%s` %d unread count: %d>1, not marking as read\n", name, chatId, chat.UnreadCount))
		return
	}
	//fmt.Printf("Chat `%s` %d unread count: %d, marking join as read\n", name, chatId, chat.UnreadCount))

	err = t.markAsRead(chatId, messageId)
	if err != nil {
		fmt.Printf("Cannot mark as read chat %d, message %d: %s\n", chatId, messageId, err.Error())

		return
	}
	chat, err = t.GetChat(chatId, true)
	if err != nil {
		fmt.Printf("Cannot get NEW unread count because chat %d not found: %s\n", chatId, err.Error())

		return
	}
	//fmt.Printf("NEW Chat `%s` %d unread count: %d\n", name, chatId, chat.UnreadCount))
}

func (t *TdApi) GetSenderName(sender client.MessageSender) string {
	chat, err := t.GetSenderObj(sender)
	if err != nil {

		return err.Error()
	}
	if sender.MessageSenderType() == "messageSenderChat" {
		name := fmt.Sprintf("%s", chat.(*client.Chat).Title)
		if name == "" {
			name = fmt.Sprintf("no_name %d", chat.(*client.Chat).Id)
		}
		return name
	} else if sender.MessageSenderType() == "messageSenderUser" {
		user := chat.(*client.User)
		return GetUserFullname(user)
	}

	return "unkown_chattype"
}

func (t *TdApi) GetSenderObj(sender client.MessageSender) (interface{}, error) {
	if sender.MessageSenderType() == "messageSenderChat" {
		chatId := sender.(*client.MessageSenderChat).ChatId
		chat, err := t.GetChat(chatId, false)
		if err != nil {
			log.Printf("Failed to request sender chat info by id %d: %s", chatId, err)

			return nil, errors.New("unknown chat")
		}

		return chat, nil
	} else if sender.MessageSenderType() == "messageSenderUser" {
		userId := sender.(*client.MessageSenderUser).UserId
		user, err := t.GetUser(userId)
		if err != nil {
			log.Printf("Failed to request user info by id %d: %s", userId, err)

			return nil, errors.New("unknown user")
		}

		return user, nil
	}

	return nil, errors.New("unknown sender type")
}

func (t *TdApi) GetChatName(chatId int64) string {
	fullChat, err := t.GetChat(chatId, false)
	if err != nil {
		log.Printf("Failed to get chat name by id %d: %s", chatId, err)

		return "no_title"
	}
	name := fmt.Sprintf("%s", fullChat.Title)
	if name == "" {
		name = fmt.Sprintf("no_name %d", chatId)
	}

	return name
}

func (t *TdApi) GetChatUsername(chatId int64) string {
	chat, err := t.GetChat(chatId, false)
	if err != nil {
		log.Printf("Failed to get chat name by id %d: %s", chatId, err)

		return ""
	}
	switch chat.Type.ChatTypeType() {
	case client.TypeChatTypeSupergroup:
		typ := chat.Type.(*client.ChatTypeSupergroup)
		sg, err := t.GetSuperGroup(typ.SupergroupId)
		if err != nil {
			log.Printf("GetChatUsername error: %s", err.Error())
			return ""
		}
		return GetUsername(sg.Usernames)
	case client.TypeChatTypePrivate:
		typ := chat.Type.(*client.ChatTypePrivate)
		user, err := t.GetUser(typ.UserId)
		if err != nil {
			log.Printf("GetChatUsername error: %s", err.Error())
			return ""
		}
		return GetUsername(user.Usernames)
	}

	return ""
}

func (t *TdApi) LoadChatsList(listId int32) {
	var chatList client.ChatList
	d, err := t.db.DeleteChatFolder(listId)
	if err != nil {
		log.Printf("failed to delete chats by list %d: %s\n", listId, err.Error())
	} else {
		log.Printf("deleted %d chats by listid %d because refresh was called\n", d, listId)
	}

	switch listId {
	case consts.ClMain:
		chatList = &client.ChatListMain{}
		log.Printf("requesting LoadChats for main list: %s", chatList.ChatListType())
	case consts.ClArchive:
		chatList = &client.ChatListArchive{}
		log.Printf("requesting LoadChats for archive: %s", chatList.ChatListType())
	default:
		chatList = &client.ChatListFolder{ChatFolderId: listId}
		log.Printf("requesting LoadChats for folder: %d", chatList.(*client.ChatListFolder).ChatFolderId)
	}

	loadChatsReq := &client.LoadChatsRequest{ChatList: chatList, Limit: 500}
	_, err = t.tdlibClient.LoadChats(loadChatsReq)

	if err == nil {
		log.Printf("load chats ok, chat list updated will be received asynchronous")
		//everything ok
		return
	}
	if err.Error() != "404 Not Found" {
		//dunno what to do yet
		log.Fatalf("[ERROR] LoadChats: %s", err)
	}
	//@see https://github.com/tdlib/td/blob/fb39e5d74667db915a75a5e58065c59af8e7d8d6/td/generate/scheme/td_api.tl#L4171

	log.Printf("all chats already loaded, trying to get them")
	getChatsReq := &client.GetChatsRequest{ChatList: chatList, Limit: 500}
	chats, err := t.tdlibClient.GetChats(getChatsReq)
	if err != nil {
		log.Fatalf("failed to get loaded chats: %s", err.Error())
	}
	for _, chat := range chats.ChatIds {
		//log.Printf("saving chat %d", chat)
		t.db.SaveChatPosition(chat, &client.ChatPosition{
			List:     chatList,
			Order:    0,
			IsPinned: false,
			Source:   nil,
		})
	}
}

func (t *TdApi) SaveChatFilters(chatFoldersUpdate *client.UpdateChatFolders) {
	log.Printf("Chat filters update! %s", chatFoldersUpdate.Type)
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
		log.Printf("New chat folder: id: %d, n: %s", folderInfo.Id, folderInfo.Name.Text.Text)

		wg.Add(1)
		go func(folderInfo *client.ChatFolderInfo, wg *sync.WaitGroup) {
			defer wg.Done()
			chatFolder, err := t.getChatFolder(folderInfo.Id)
			if err != nil {
				log.Printf("Failed to load chat folder: id: %d, n: %s, reason: %s", folderInfo.Id, folderInfo.Name.Text.Text, err.Error())

				return
			}
			t.db.SaveChatFolder(chatFolder, folderInfo)
			log.Printf("Chat folder LOADED: id: %d, n: %s", folderInfo.Id, folderInfo.Name.Text.Text)
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
		log.Printf("Deleted chat folder: id: %d, n: %s", existningFolder.Id, existningFolder.Title)
	}

	t.chatFolders = t.db.LoadChatFolders()
}

func (t *TdApi) SaveChatAddedToList(upd *client.UpdateChatAddedToList) {
	//j, _ := json.Marshal(upd)
	//log.Printf("saving chat added to list %s : %s", string(j))
	switch upd.ChatList.ChatListType() {
	case client.TypeChatListMain:
		position := client.ChatPosition{
			List:     &client.ChatListMain{},
			Order:    0,
			IsPinned: false,
			Source:   nil,
		}
		t.db.SaveChatPosition(upd.ChatId, &position)
	case client.TypeChatListArchive:
		position := client.ChatPosition{
			List:     &client.ChatListArchive{},
			Order:    0,
			IsPinned: false,
			Source:   nil,
		}
		t.db.SaveChatPosition(upd.ChatId, &position)
	case client.TypeChatListFolder:
		position := client.ChatPosition{
			List:     &client.ChatListFolder{ChatFolderId: upd.ChatList.(*client.ChatListFolder).ChatFolderId},
			Order:    0,
			IsPinned: false,
			Source:   nil,
		}
		t.db.SaveChatPosition(upd.ChatId, &position)
	default:
		log.Printf("unknown chatlist type: %s", upd.ChatList.ChatListType())
	}
}

func (t *TdApi) RemoveChatRemovedFromList(upd *client.UpdateChatRemovedFromList) {
	j, _ := json.Marshal(upd)
	log.Printf("NOT IMPLEMENTED: removing chat removed from list %s : %s", string(j))
	return
	switch upd.ChatList.ChatListType() {
	case client.TypeChatListMain:
		position := client.ChatPosition{
			List:     &client.ChatListMain{},
			Order:    0,
			IsPinned: false,
			Source:   nil,
		}
		t.db.SaveChatPosition(upd.ChatId, &position)
	case client.TypeChatListArchive:
		position := client.ChatPosition{
			List:     &client.ChatListArchive{},
			Order:    0,
			IsPinned: false,
			Source:   nil,
		}
		t.db.SaveChatPosition(upd.ChatId, &position)
	case client.TypeChatListFolder:
		position := client.ChatPosition{
			List:     &client.ChatListFolder{ChatFolderId: upd.ChatList.(*client.ChatListFolder).ChatFolderId},
			Order:    0,
			IsPinned: false,
			Source:   nil,
		}
		t.db.SaveChatPosition(upd.ChatId, &position)
	default:
		log.Printf("unknown chatlist type: %s", upd.ChatList.ChatListType())
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

	return t.tdlibClient.GetOptionAsync(&req)
}

func (t *TdApi) GetActiveSessions() (*client.Sessions, error) {

	return t.tdlibClient.GetActiveSessions()
}

func (t *TdApi) GetChatHistory(chatId int64, lastId int64) (*client.Messages, error) {
	req := &client.GetChatHistoryRequest{ChatId: chatId, Limit: 100, FromMessageId: lastId, Offset: 0}

	return t.tdlibClient.GetChatHistory(req)
}

func (t *TdApi) DeleteMessages(chatId int64, messageIds []int64) (*client.Ok, error) {
	req := &client.DeleteMessagesRequest{ChatId: chatId, MessageIds: messageIds}

	return t.tdlibClient.DeleteMessages(req)
}

func (t *TdApi) GetChatMember(chatId int64) (*client.ChatMember, error) {
	m := client.MessageSenderUser{UserId: t.dbData.Id}
	req := &client.GetChatMemberRequest{ChatId: chatId, MemberId: &m}

	return t.tdlibClient.GetChatMember(req)
}

func (t *TdApi) GetScheduledMessages(chatId int64) (*client.Messages, error) {
	req := &client.GetChatScheduledMessagesRequest{ChatId: chatId}

	return t.tdlibClient.GetChatScheduledMessages(req)
}

func (t *TdApi) ScheduleForwardedMessage(targetChatId int64, fromChatId int64, messageIds []int64, sendAtDate int32, sendCopy bool) (*client.Messages, error) {
	opts := &client.MessageSendOptions{SchedulingState: &client.MessageSchedulingStateSendAtDate{SendDate: sendAtDate}}
	req := &client.ForwardMessagesRequest{ChatId: targetChatId, MessageIds: messageIds, FromChatId: fromChatId, Options: opts, SendCopy: sendCopy}

	res, err := t.tdlibClient.ForwardMessages(req)
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

func (t *TdApi) EditMessageSchedulingState(chatId int64, messageId int64, schedulingStateType string, sendDate int32) (*client.Ok, error) {
	var schedulingState client.MessageSchedulingState
	switch schedulingStateType {
	case client.TypeMessageSchedulingStateSendAtDate:
		schedulingState = &client.MessageSchedulingStateSendAtDate{SendDate: sendDate}
	case client.TypeMessageSchedulingStateSendWhenOnline:
		schedulingState = &client.MessageSchedulingStateSendWhenOnline{}
	}

	req := &client.EditMessageSchedulingStateRequest{
		ChatId:          chatId,
		MessageId:       messageId,
		SchedulingState: schedulingState,
	}

	return t.tdlibClient.EditMessageSchedulingState(req)
}

func (t *TdApi) GetStorage() TdStorageInterface {
	//@TODO: mutex?
	return t.db
}

func (t *TdApi) Close() {

	t.tdlibClient.Close()
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
