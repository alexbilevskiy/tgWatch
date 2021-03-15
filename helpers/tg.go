package helpers

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"go-tdlib/client"
	"log"
	"path/filepath"
	"strconv"
	"tgWatch/config"
	"tgWatch/structs"
	"time"
)

func initTdlib() {
	LoadChatFilters()
	localChats = make(map[int64]*client.Chat)
	authorizer := client.ClientAuthorizer()
	go client.CliInteractor(authorizer)

	authorizer.TdlibParameters <- &client.TdlibParameters{
		UseTestDc:              false,
		DatabaseDirectory:      filepath.Join(".tdlib", "database"),
		FilesDirectory:         filepath.Join(".tdlib", "files"),
		UseFileDatabase:        true,
		UseChatInfoDatabase:    true,
		UseMessageDatabase:     true,
		UseSecretChats:         false,
		ApiId:                  config.Config.ApiId,
		ApiHash:                config.Config.ApiHash,
		SystemLanguageCode:     "en",
		DeviceModel:            "Linux",
		SystemVersion:          "1.0.0",
		ApplicationVersion:     "1.0.0",
		EnableStorageOptimizer: true,
		IgnoreFileNames:        false,
	}

	logVerbosity := client.WithLogVerbosity(&client.SetLogVerbosityLevelRequest{
		NewVerbosityLevel: 0,
	})

	var err error
	tdlibClient, err = client.NewClient(authorizer, logVerbosity)
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

	log.Printf("Me: %s %s [%s]", me.FirstName, me.LastName, me.Username)
}

func ListenUpdates()  {
	listener := tdlibClient.GetListener()
	defer listener.Close()

	for update := range listener.Updates {
		if update.GetClass() == client.ClassUpdate {
			t := update.GetType()
			switch t {
			case "updateChatActionBar":
			case "updateChatIsBlocked":
			case "updateChatDraftMessage":
			case "updateUserStatus":
			case "updateChatReadInbox":
			case "updateChatReadOutbox":
			case "updateUnreadMessageCount":
			case "updateUnreadChatCount":
			case "updateMessageInteractionInfo":
			case "updateChatReplyMarkup":
			case "updateChatPermissions":
			case "updateChatNotificationSettings":
			case "updateChatUnreadMentionCount":
			case "updateMessageMentionRead":
			case "updateMessageIsPinned":
			case "updateChatHasScheduledMessages":
			case "updateHavePendingNotifications":
			case "updateRecentStickers":

			case "updateSupergroup":
			case "updateSupergroupFullInfo":
			case "updateBasicGroup":
			case "updateBasicGroupFullInfo":
			case "updateUser":
			case "updateUserFullInfo":
			case "updateChatPhoto":
				break

			case "updateChatTitle":
				upd := update.(*client.UpdateChatTitle)
				log.Printf("Renamed chat id:%d to `%s`", upd.ChatId, upd.Title)

				break
			case "updateNewChat":
				upd := update.(*client.UpdateNewChat)
				localChats[upd.Chat.Id] = upd.Chat
				//@TODO: store chat to local cache, avoid using getChat request
				DLog(fmt.Sprintf("New chat added: %d / %s", upd.Chat.Id, upd.Chat.Title))
				if len(upd.Chat.Positions) > 0 {
					saveChatPosition(upd.Chat.Id, upd.Chat.Positions[0])
				}

				break
			case "updateConnectionState":
				upd := update.(*client.UpdateConnectionState)
				log.Printf("Connection state changed: %s", upd.State.ConnectionStateType())

				break
			case "updateUserChatAction":
				upd := update.(*client.UpdateUserChatAction)
				if upd.ChatId < 0 {
					DLog(fmt.Sprintf("Skipping action in non-user chat %d: %s", upd.ChatId, upd.Action.ChatActionType()))
					break
				}
				user, err := GetUser(upd.UserId)
				userName := "err_name"
				if err != nil {
					fmt.Printf("failed to get user %d: %s", upd.UserId, err)
				} else {
					userName = getUserFullname(user)
				}
				log.Printf("User action in chat `%s`: id: %d, name: `%s`, action: %s", GetChatName(upd.ChatId), upd.UserId, userName, upd.Action.ChatActionType())

				break
			case "updateChatLastMessage":
				upd := update.(*client.UpdateChatLastMessage)
				if len(upd.Positions) == 0 {
					break
				}
				saveChatPosition(upd.ChatId, upd.Positions[0])

				break
			case "updateOption":
				upd := update.(*client.UpdateOption)
				log.Printf("Update option %s: %s", upd.Name, JsonMarshalStr(upd.Value))

				break
			case "updateChatPosition":
				upd := update.(*client.UpdateChatPosition)
				saveChatPosition(upd.ChatId, upd.Position)

				break
			case "updateChatFilters":
				upd := update.(*client.UpdateChatFilters)
				SaveChatFilters(upd)

				break

			case "updateDeleteMessages":
				upd := update.(*client.UpdateDeleteMessages)
				if !upd.IsPermanent || upd.FromCache {

					break
				}
				if checkSkippedChat(strconv.FormatInt(upd.ChatId, 10)) || checkChatFilter(upd.ChatId) {

					break
				}

				skipUpdate := 0
				for _, messageId := range upd.MessageIds {
					savedMessage, err := FindUpdateNewMessage(upd.ChatId, messageId)
					if err != nil {

						continue
					}
					if checkSkippedChat(strconv.FormatInt(GetChatIdBySender(savedMessage.Message.Sender), 10)) {
						DLog(fmt.Sprintf("Skip deleted message %d from sender %d, `%s`", messageId, GetChatIdBySender(savedMessage.Message.Sender), GetSenderName(savedMessage.Message.Sender)))
						skipUpdate++

						continue
					}
					if savedMessage.Message.Content == nil {
						log.Printf("Skip deleted message %d with unknown content from %s", messageId, GetChatIdBySender(savedMessage.Message.Sender))

						continue
					}
					if savedMessage.Message.Content.MessageContentType() == "messageChatAddMembers" {
						DLog(fmt.Sprintf("Skip deleted message %d (chat join of user %d)", messageId, GetChatIdBySender(savedMessage.Message.Sender)))
						skipUpdate++

						continue
					}
				}
				if skipUpdate == len(upd.MessageIds) {

					break
				}
				mongoId := SaveUpdate(t, upd, 0)

				chatName := GetChatName(upd.ChatId)
				intLink := fmt.Sprintf("http://%s/d/%d/%s", config.Config.WebListen, upd.ChatId, ImplodeInt(upd.MessageIds))
				count := len(upd.MessageIds)
				DLog(fmt.Sprintf("[%s] DELETED %d Messages from chat: %d, `%s`, %s", mongoId, count, upd.ChatId, chatName, intLink))

				break

			case "updateNewMessage":
				upd := update.(*client.UpdateNewMessage)
				if checkSkippedChat(strconv.FormatInt(upd.Message.ChatId, 10)) || checkChatFilter(upd.Message.ChatId) {

					break
				}
				//senderChatId := GetChatIdBySender(upd.Message.Sender)
				SaveUpdate(t, upd, upd.Message.Date)
				//mongoId := SaveUpdate(t, upd, upd.Message.Date)
				//link := GetLink(tdlibClient, upd.Message.ChatId, upd.Message.Id)
				//chatName := GetChatName(upd.Message.ChatId)
				//intLink := fmt.Sprintf("http://%s/e/%d/%d", config.Config.WebListen, upd.Message.ChatId, upd.Message.Id)
				//log.Printf("[%s] New Message from chat: %d, `%s`, %s, %s", mongoId, upd.Message.ChatId, chatName, link, intLink)

				break
			case "updateMessageEdited":
				upd := update.(*client.UpdateMessageEdited)
				if checkSkippedChat(strconv.FormatInt(upd.ChatId, 10)) || checkChatFilter(upd.ChatId) {

					break
				}

				if upd.ReplyMarkup != nil {
					//messages with buttons - reactions, likes etc

					break
				}
				SaveUpdate(t, upd, upd.EditDate)
				//mongoId := SaveUpdate(t, upd, upd.EditDate)
				//link := GetLink(tdlibClient, upd.ChatId, upd.MessageId)
				//chatName := GetChatName(upd.ChatId)
				//intLink := fmt.Sprintf("http://%s/e/%d/%d", config.Config.WebListen, upd.ChatId, upd.MessageId)
				//log.Printf("[%s] EDITED msg! Chat: %d, msg %d, `%s`, %s, %s", mongoId, upd.ChatId, upd.MessageId, chatName, link, intLink)

				break
			case "updateMessageContent":
				upd := update.(*client.UpdateMessageContent)
				if checkSkippedChat(strconv.FormatInt(upd.ChatId, 10)) || checkChatFilter(upd.ChatId) {

					break
				}
				if upd.NewContent.MessageContentType() == "messagePoll" {
					//dont save "poll" updates - that's just counters, users cannot update polls manually
					break
				}
				mongoId := SaveUpdate(t, upd, 0)

				link := GetLink(upd.ChatId, upd.MessageId)
				chatName := GetChatName(upd.ChatId)
				intLink := fmt.Sprintf("http://%s/e/%d/%d", config.Config.WebListen, upd.ChatId, upd.MessageId)
				DLog(fmt.Sprintf("[%s] EDITED content! Chat: %d, msg %d, %s, %s, %s", mongoId, upd.ChatId, upd.MessageId, chatName, link, intLink))

				break
			case "updateFile":
				upd := update.(*client.UpdateFile)
				if upd.File.Local.IsDownloadingActive {
					DLog(fmt.Sprintf("File downloading: %d/%d bytes", upd.File.Local.DownloadedSize, upd.File.ExpectedSize))
				} else {
					DLog(fmt.Sprintf("File downloaded: %d bytes, path: %s", upd.File.Local.DownloadedSize, upd.File.Local.Path))
				}

				break
			default:
				j, _ := json.Marshal(update)
				log.Printf("Unknown update %s : %s", t, string(j))
			}
		}
	}
}

func GetChatIdBySender(sender client.MessageSender) int64 {
	senderChatId := int64(0)
	if sender.MessageSenderType() == "messageSenderChat" {
		senderChatId = sender.(*client.MessageSenderChat).ChatId
	} else if sender.MessageSenderType() == "messageSenderUser" {
		senderChatId = int64(sender.(*client.MessageSenderUser).UserId)
	}

	return senderChatId
}

func GetSenderName(sender client.MessageSender) string {
	chat, err := GetSenderObj(sender)
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
		return getUserFullname(user)
	}

	return "unkown_chattype"
}

func getUserFullname(user *client.User) string {
	name := ""
	if user.FirstName != "" {
		name = user.FirstName
	}
	if user.LastName != "" {
		name = fmt.Sprintf("%s %s", name, user.LastName)
	}
	if user.Username != "" {
		name = fmt.Sprintf("%s (@%s)", name, user.Username)
	}
	if name == "" {
		name = fmt.Sprintf("no_name %d", user.Id)
	}
	return name
}

func GetSenderObj(sender client.MessageSender) (interface{}, error) {
	if sender.MessageSenderType() == "messageSenderChat" {
		chatId := sender.(*client.MessageSenderChat).ChatId
		chat, err := GetChat(chatId)
		if err != nil {
			log.Printf("Failed to request sender chat info by id %d: %s", chatId, err)

			return nil, errors.New("unknown chat")
		}

		return chat, nil
	} else if sender.MessageSenderType() == "messageSenderUser" {
		userId := sender.(*client.MessageSenderUser).UserId
		user, err := GetUser(userId)
		if err != nil {
			log.Printf("Failed to request user info by id %d: %s", userId, err)

			return nil, errors.New("unknown user")
		}

		return user, nil
	}

	return nil, errors.New("unknown sender type")
}

func GetLink(chatId int64, messageId int64) string {
	linkReq := &client.GetMessageLinkRequest{ChatId: chatId, MessageId: messageId}
	link, err := tdlibClient.GetMessageLink(linkReq)
	if err != nil {
		if err.Error() != "400 Public message links are available only for messages in supergroups and channel chats" {
			log.Printf("Failed to get msg link by chat id %d, msg id %d: %s", chatId, messageId, err)
		}

		return ""
	}

	return link.Link
}

func GetChatName(chatId int64) string {
	fullChat, err := GetChat(chatId)
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

func GetChat(chatId int64) (*client.Chat, error) {
	fullChat, ok := localChats[chatId]
	if ok {
		//fmt.Printf("Found local chat %d\n", chatId)

		return fullChat, nil
	}
	req := &client.GetChatRequest{ChatId: chatId}
	fullChat, err := tdlibClient.GetChat(req)
	if err != nil {
		fmt.Printf("Caching local chat %d\n", chatId)
		localChats[chatId] = fullChat
	}

	return fullChat, err
}
func GetUser(userId int32) (*client.User, error) {
	userReq := &client.GetUserRequest{UserId: userId}

	return tdlibClient.GetUser(userReq)
}

func GetContent(content client.MessageContent) string {
	if content == nil {

		return "UNSUPPORTED_CONTENT"
	}
	cType := content.MessageContentType()
	switch cType {
	case "messageText":
		msg := content.(*client.MessageText)

		return fmt.Sprintf("%s", msg.Text.Text)
	case "messagePhoto":
		msg := content.(*client.MessagePhoto)

		return fmt.Sprintf("Photo, %s", msg.Caption.Text)
	case "messageVideo":
		msg := content.(*client.MessageVideo)

		return fmt.Sprintf("Video, %s", msg.Caption.Text)
	case "messageAnimation":
		msg := content.(*client.MessageAnimation)

		return fmt.Sprintf("GIF, %s", msg.Caption.Text)
	case "messagePoll":
		msg := content.(*client.MessagePoll)

		return fmt.Sprintf("Poll, %s", msg.Poll.Question)
	case "messageSticker":
		msg := content.(*client.MessageSticker)

		return fmt.Sprintf("Sticker, %s", msg.Sticker.Emoji)
	default:

		return JsonMarshalStr(content)
	}
}

func DownloadFile(id int32) (*client.File, error) {
	req := client.DownloadFileRequest{FileId: id, Priority: 1, Synchronous: true}
	file, err := tdlibClient.DownloadFile(&req)
	if err != nil {
		log.Printf("Cannot download file: %s %d", err, id)

		return nil, err
	}

	return file, nil
}

func DownloadFileByRemoteId(id string) (*client.File, error) {
	remoteFileReq := client.GetRemoteFileRequest{RemoteFileId: id}
	remoteFile, err := tdlibClient.GetRemoteFile(&remoteFileReq)
	if err != nil {
		log.Printf("Cannot download remote file: %s %s", err, id)

		return nil, err
	}

	return DownloadFile(remoteFile.Id)
}

func GetContentStructs(content client.MessageContent) []structs.MessageAttachment {
	if content == nil {

		return nil
	}
	cType := content.MessageContentType()
	var cnt []structs.MessageAttachment
	switch cType {
	case "messageText":

		return nil
	case "messagePhoto":
		msg := content.(*client.MessagePhoto)
		s := structs.MessageAttachment{
			T: msg.Photo.Type,
			Id: msg.Photo.Sizes[len(msg.Photo.Sizes)-1].Photo.Remote.Id,
			Thumb: base64.StdEncoding.EncodeToString(msg.Photo.Minithumbnail.Data),
		}
		for _, size := range msg.Photo.Sizes {
			s.Link = append(s.Link, fmt.Sprintf("http://%s/f/%s", config.Config.WebListen, size.Photo.Remote.Id))
		}
		cnt = append(cnt, s)

		return cnt
	case "messageVideo":
		msg := content.(*client.MessageVideo)
		s := structs.MessageAttachment{
			T: msg.Video.Type,
			Id: msg.Video.Video.Remote.Id,
			Link: append(make([]string, 0), fmt.Sprintf("http://%s/f/%s", config.Config.WebListen, msg.Video.Video.Remote.Id)),
			Thumb: base64.StdEncoding.EncodeToString(msg.Video.Minithumbnail.Data),
		}
		cnt = append(cnt, s)

		return cnt
	case "messageAnimation":
		msg := content.(*client.MessageAnimation)
		s := structs.MessageAttachment{
			T: msg.Animation.Type,
			Id: msg.Animation.Animation.Remote.Id,
			Link: append(make([]string, 0), fmt.Sprintf("http://%s/f/%s", config.Config.WebListen, msg.Animation.Animation.Remote.Id)),
			Thumb: base64.StdEncoding.EncodeToString(msg.Animation.Minithumbnail.Data),
		}

		cnt = append(cnt, s)

		return cnt
	case "messagePoll":
		//msg := content.(*client.MessagePoll)

		return nil
	case "messageSticker":
		msg := content.(*client.MessageSticker)
		s := structs.MessageAttachment{
			T: msg.Sticker.Type,
			Id: msg.Sticker.Sticker.Remote.Id,
			Link: append(make([]string, 0), fmt.Sprintf("http://%s/f/%s", config.Config.WebListen, msg.Sticker.Sticker.Remote.Id)),
		}
		cnt = append(cnt, s)

		return cnt
	default:
		log.Printf("Unknown content type: %s", cType)

		return nil
	}
}

func getChatsList() []*client.Chat {
	maxChatId := client.JsonInt64(int64((^uint64(0)) >> 1))
	offsetOrder := maxChatId
	log.Printf("Requesting chats with max id: %d", maxChatId)

	var chatList []*client.Chat
	page := 0
	offsetChatId := int64(0)
	for {
		log.Printf("GetChats requesting page %d, offset %d", page, offsetChatId)
		chatsRequest := &client.GetChatsRequest{OffsetOrder: offsetOrder, Limit: 10, OffsetChatId: offsetChatId}
		chats, err := tdlibClient.GetChats(chatsRequest)
		if err != nil {
			log.Fatalf("[ERROR] GetChats: %s", err)
		}
		log.Printf("GetChats got page %d with %d chats", page, chats.TotalCount)
		for _, chatId := range chats.ChatIds {
			log.Printf("New ChatID %d", chatId)
			chatRequest := &client.GetChatRequest{ChatId: chatId}
			chat, err := tdlibClient.GetChat(chatRequest)
			if err != nil {
				log.Printf("[ERROR] GetChat id %d: %s", chatId, err)

				continue
			}
			log.Printf("Got chatID %d, position %d, title `%s`", chatId, chat.Positions[0].Order, chat.Title)
			saveChatPosition(chatId, chat.Positions[0])
			offsetChatId = chat.Id
			offsetOrder = chat.Positions[0].Order

			chatList = append(chatList, chat)
		}

		if len(chats.ChatIds) == 0 {
			log.Printf("Reached end of the list")

			break
		}
		time.Sleep(1 * time.Second)
		page++
		log.Println()
	}

	return chatList
}

func checkSkippedChat(chatId string) bool {
	if _, ok := config.Config.IgnoreAuthorIds[chatId]; ok {

		return true
	}
	if _, ok := config.Config.IgnoreChatIds[chatId]; ok {

		return true
	}

	return false
}

func checkChatFilter(chatId int64) bool {
	for _, filter := range chatFilters {
		for _, chatInFilter := range filter.IncludedChats {
			if chatInFilter == chatId && config.Config.IgnoreFolders[filter.Title] {
				//log.Printf("Skip chat %d because it's in skipped folder %s", chatId, filter.Title)

				return true
			}
		}
	}

	return false
}