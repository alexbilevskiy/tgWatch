package libs

import (
	"fmt"
	"go-tdlib/client"
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
		log.Printf("Cannot download file: %s %d", err, id)

		return nil, err
	}

	return file, nil
}

func DownloadFileByRemoteId(acc int64, id string) (*client.File, error) {
	remoteFileReq := client.GetRemoteFileRequest{RemoteFileId: id}
	remoteFile, err := tdlibClient[acc].GetRemoteFile(&remoteFileReq)
	if err != nil {
		log.Printf("Cannot download remote file: %s %s", err, id)

		return nil, err
	}

	return DownloadFile(acc, remoteFile.Id)
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
		log.Printf("Failed to get msg link by chat id %d, msg id %d: %s", chatId, messageId, err)

		return ""
	}

	return link.Link
}

func loadChats(acc int64, chatList client.ChatList) error {
	chatsRequest := &client.LoadChatsRequest{ChatList: chatList, Limit: 500}
	_, err := tdlibClient[acc].LoadChats(chatsRequest)

	return err
}

func getChatFilter(acc int64, filterId int32) (*client.ChatFilter, error) {
	req := &client.GetChatFilterRequest{ChatFilterId: filterId}
	return tdlibClient[acc].GetChatFilter(req)
}
