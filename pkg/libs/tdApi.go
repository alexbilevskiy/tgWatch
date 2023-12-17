package libs

import (
	"errors"
	"fmt"
	"github.com/zelenin/go-tdlib/client"
	"log"
	"sync"
)

var m = sync.RWMutex{}

func GetChat(acc int64, chatId int64, force bool) (*client.Chat, error) {
	m.RLock()
	fullChat, ok := localChats[acc][chatId]
	m.RUnlock()
	if !force && ok {

		return fullChat, nil
	}
	req := &client.GetChatRequest{ChatId: chatId}
	fullChat, err := tdlibClient[acc].GetChat(req)
	if err == nil {
		DLog(fmt.Sprintf("Caching local chat %d\n", chatId))
		CacheChat(acc, fullChat)
	}

	return fullChat, err
}

func CacheChat(acc int64, chat *client.Chat) {
	m.Lock()
	localChats[acc][chat.Id] = chat
	m.Unlock()
}

func GetUser(acc int64, userId int64) (*client.User, error) {
	userReq := &client.GetUserRequest{UserId: userId}

	return tdlibClient[acc].GetUser(userReq)
}

func GetSuperGroup(acc int64, sgId int64) (*client.Supergroup, error) {
	sgReq := &client.GetSupergroupRequest{SupergroupId: sgId}

	return tdlibClient[acc].GetSupergroup(sgReq)
}

func GetBasicGroup(acc int64, groupId int64) (*client.BasicGroup, error) {
	bgReq := &client.GetBasicGroupRequest{BasicGroupId: groupId}

	return tdlibClient[acc].GetBasicGroup(bgReq)
}

func DownloadFile(acc int64, id int32) (*client.File, error) {
	req := client.DownloadFileRequest{FileId: id, Priority: 1, Synchronous: true}
	file, err := tdlibClient[acc].DownloadFile(&req)
	if err != nil {
		//log.Printf("Cannot download file: %s %d", err, id)

		return nil, errors.New("downloading error: " + err.Error())
	}

	return file, nil
}

func DownloadFileByRemoteId(acc int64, id string) (*client.File, error) {
	remoteFileReq := client.GetRemoteFileRequest{RemoteFileId: id}
	remoteFile, err := tdlibClient[acc].GetRemoteFile(&remoteFileReq)
	if err != nil {
		//log.Printf("cannot get remote file info: %s %s", err, id)

		return nil, errors.New("remoteFile request error: " + err.Error())
	}
	//if remoteFile.Local.IsDownloadingCompleted {
	//	log.Printf("Not dowloading file again: %s", remoteFile.Local.Path)
	//
	//	return remoteFile, nil
	//}

	return DownloadFile(acc, remoteFile.Id)
}

func GetCustomEmoji(customEmojisIds []client.JsonInt64) (*client.Stickers, error) {
	customEmojisReq := client.GetCustomEmojiStickersRequest{CustomEmojiIds: customEmojisIds}
	customEmojis, err := tdlibClient[currentAcc].GetCustomEmojiStickers(&customEmojisReq)
	if err != nil {
		return nil, errors.New("custom emoji error: " + err.Error())
	}

	return customEmojis, nil
}

func markAsRead(acc int64, chatId int64, messageId int64) error {
	req := &client.ViewMessagesRequest{ChatId: chatId, MessageIds: append(make([]int64, 0), messageId), ForceRead: true}
	_, err := tdlibClient[acc].ViewMessages(req)

	return err
}

func GetLink(acc int64, chatId int64, messageId int64) string {
	chat, err := GetChat(acc, chatId, false)
	if err != nil {
		log.Printf("GetLink: chat %d not found: %s", chatId, err.Error())
		return ""
	}
	if chat.Type.ChatTypeType() != client.TypeChatTypeSupergroup {
		DLog(fmt.Sprintf("GetLink: not available for chat `%s` (%d) with type %s", chat.Title, chatId, chat.Type.ChatTypeType()))
		return ""
	}

	linkReq := &client.GetMessageLinkRequest{ChatId: chatId, MessageId: messageId}
	link, err := tdlibClient[acc].GetMessageLink(linkReq)
	if err != nil {
		if err.Error() != "400 Message not found" {
			log.Printf("Failed to get msg link by chat id %d, msg id %d: %s", chatId, messageId, err)
		}

		return ""
	}

	return link.Link
}

func SendMessage(acc int64, text string, chatId int64, replyToMessageId *int64) {
	mtext := &client.FormattedText{Text: text}
	content := &client.InputMessageText{Text: mtext}
	var req *client.SendMessageRequest
	if replyToMessageId == nil {
		req = &client.SendMessageRequest{ChatId: chatId, InputMessageContent: content}
	} else {
		replyTo := client.InputMessageReplyToMessage{MessageId: *replyToMessageId}
		req = &client.SendMessageRequest{ChatId: chatId, ReplyTo: &replyTo, InputMessageContent: content}
	}
	message, err := tdlibClient[acc].SendMessage(req)
	if err != nil {
		log.Printf("Failed to send message to chat %d: %s", chatId, err.Error())
	} else {
		log.Printf("Sent message to chat %d! new (virtual) message id: %d", chatId, message.Id)
	}
}

func GetLinkInfo(acc int64, link string) (client.InternalLinkType, interface{}, error) {
	linkTypeReq := &client.GetInternalLinkTypeRequest{Link: link}
	linkType, err := tdlibClient[acc].GetInternalLinkType(linkTypeReq)
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("get link type error: %s", err.Error()))
	}
	switch linkType.InternalLinkTypeType() {
	case client.TypeInternalLinkTypeMessage:
		linkType := linkType.(*client.InternalLinkTypeMessage)
		messageLinkInfoReq := &client.GetMessageLinkInfoRequest{Url: link}
		messageLinkInfo, err := tdlibClient[acc].GetMessageLinkInfo(messageLinkInfoReq)
		if err == nil {
			return linkType, messageLinkInfo.Message, nil
		}

		return linkType, err, nil

	case client.TypeInternalLinkTypePublicChat:
		linkType := linkType.(*client.InternalLinkTypePublicChat)
		publicChatReq := &client.SearchPublicChatRequest{Username: linkType.ChatUsername}
		publicChat, err := tdlibClient[acc].SearchPublicChat(publicChatReq)
		if err == nil {
			return linkType, publicChat, nil
		}

		return linkType, err, nil

	case client.TypeInternalLinkTypeChatInvite:
		linkType := linkType.(*client.InternalLinkTypeChatInvite)
		chatInviteLinkReq := &client.CheckChatInviteLinkRequest{InviteLink: linkType.InviteLink}
		chatInviteLink, err := tdlibClient[acc].CheckChatInviteLink(chatInviteLinkReq)
		if err == nil {
			return linkType, chatInviteLink, nil
		}

		return linkType, err, nil

	default:
		return linkType, errors.New(fmt.Sprintf("unknown link type: %s", linkType.InternalLinkTypeType())), nil
	}
}

func GetMessage(acc int64, chatId int64, messageId int64) (*client.Message, error) {
	log.Printf("get message %d/%d", chatId, messageId)
	var err error

	openChatReq := &client.OpenChatRequest{ChatId: chatId}
	_, err = tdlibClient[acc].OpenChat(openChatReq)
	if err != nil {

		return nil, errors.New(fmt.Sprintf("open chat error: %s", err.Error()))
	}

	messageIds := make([]int64, 0)
	messageIds = append(messageIds, messageId)
	viewMessagesReq := &client.ViewMessagesRequest{ChatId: chatId, MessageIds: messageIds}
	_, err = tdlibClient[acc].ViewMessages(viewMessagesReq)
	if err != nil {

		return nil, errors.New(fmt.Sprintf("failed to view message: %s", err.Error()))
	}
	//log.Printf("sleeping before get message %d/%d", chatId, messageId)
	//time.Sleep(time.Second * 5)

	getMessageReq := &client.GetMessageRequest{ChatId: chatId, MessageId: messageId}
	message, err := tdlibClient[acc].GetMessage(getMessageReq)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("get message error: %s", err.Error()))
	}

	return message, nil
}

func loadChats(acc int64, chatList client.ChatList) error {
	chatsRequest := &client.LoadChatsRequest{ChatList: chatList, Limit: 500}
	_, err := tdlibClient[acc].LoadChats(chatsRequest)

	return err
}

func getChatFolder(acc int64, folderId int32) (*client.ChatFolder, error) {
	req := &client.GetChatFolderRequest{ChatFolderId: folderId}
	return tdlibClient[acc].GetChatFolder(req)
}

func LoadChatHistory(acc int64, chatId int64, fromMessageId int64, offset int32) (*client.Messages, error) {
	chatHistoryRequest := client.GetChatHistoryRequest{ChatId: chatId, Offset: offset, FromMessageId: fromMessageId, OnlyLocal: false, Limit: 50}
	messages, err := tdlibClient[acc].GetChatHistory(&chatHistoryRequest)
	if err != nil {

		return nil, errors.New(fmt.Sprintf("failed to load history: %s", err.Error()))
	}

	return messages, nil
}
