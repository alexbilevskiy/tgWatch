package helpers

import (
	"errors"
	"fmt"
	"go-tdlib/client"
	"log"
	"path/filepath"
	"strconv"
	"tgWatch/config"
	"time"
)

func initTdlib() {
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
			case "updateUserFullInfo":
			case "updateChatActionBar":
			case "updateChatIsBlocked":
			case "updateChatPosition":

			case "updateOption":
			case "updateChatDraftMessage":
			case "updateUserStatus":
			case "updateChatReadInbox":
			case "updateChatReadOutbox":
			case "updateUnreadMessageCount":
			case "updateUnreadChatCount":
			case "updateChatLastMessage":
			case "updateUserChatAction":
			case "updateMessageInteractionInfo":
			case "updateChatReplyMarkup":
			case "updateChatPermissions":
			case "updateChatNotificationSettings":
			case "updateChatUnreadMentionCount":
			case "updateMessageMentionRead":
			case "updateConnectionState":
			case "updateMessageIsPinned":
			case "updateChatHasScheduledMessages":

			case "updateNewChat":
			case "updateHavePendingNotifications":
			case "updateSupergroupFullInfo":
			case "updateSupergroup":
			case "updateBasicGroup":
			case "updateBasicGroupFullInfo":
			case "updateChatPhoto":
			case "updateUser":
			case "updateChatTitle":
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
					savedMessage, err := FindUpdateNewMessage(messageId)
					if err != nil {

						continue
					}
					if checkSkippedChat(strconv.FormatInt(GetChatIdBySender(savedMessage.Message.Sender), 10)) {
						log.Printf("Skip deleted message %d from sender %d, `%s`", messageId, GetChatIdBySender(savedMessage.Message.Sender), GetSenderName(savedMessage.Message.Sender))
						skipUpdate++

						continue
					}
					if savedMessage.Message.Content.MessageContentType() == "messageChatAddMembers" {
						log.Printf("Skip deleted message %d (chat join of user %d)", messageId, GetChatIdBySender(savedMessage.Message.Sender))
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
				log.Printf("[%s] DELETED %d Messages from chat: %d, `%s`, %s", mongoId, count, upd.ChatId, chatName, intLink)

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

				link := GetLink(tdlibClient, upd.ChatId, upd.MessageId)
				chatName := GetChatName(upd.ChatId)
				intLink := fmt.Sprintf("http://%s/e/%d/%d", config.Config.WebListen, upd.ChatId, upd.MessageId)
				log.Printf("[%s] EDITED content! Chat: %d, msg %d, %s, %s, %s", mongoId, upd.ChatId, upd.MessageId, chatName, link, intLink)
				//log.Printf("%s", GetContent(upd.NewContent))

				break
			default:
				log.Printf("%s : %#v", t, update)
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
		return fmt.Sprintf("%s", chat.(*client.Chat).Title)
	} else if sender.MessageSenderType() == "messageSenderUser" {
		user := chat.(*client.User)
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
		return name
	}

	return "unkown_chattype"
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

func GetLink(tdlibClient *client.Client, chatId int64, messageId int64) string {
	linkReq := &client.GetMessageLinkRequest{ChatId: chatId, MessageId: messageId}
	link, err := tdlibClient.GetMessageLink(linkReq)
	if err != nil {
		if err.Error() != "400 Public message links are available only for messages in supergroups and channel chats" {
			log.Printf("Failed to get msg link by chat id %d, msg id %d: %s", chatId, messageId, err)
		}

		return "no_link"
	}

	return link.Link
}

func GetChatName(chatId int64) string {
	fullChat, err := GetChat(chatId)
	if err != nil {
		log.Printf("Failed to get chat name by id %d: %s", chatId, err)

		return "no_title"
	}

	return fmt.Sprintf("%s", fullChat.Title)
}

func GetChat(chatId int64) (*client.Chat, error) {
	req := &client.GetChatRequest{ChatId: chatId}
	//@TODO sometimes it fails on `json: cannot unmarshal object into Go struct field .last_message of type client.MessageSender`
	fullChat, err := tdlibClient.GetChat(req)

	return fullChat, err
}
func GetUser(userId int32) (*client.User, error) {
	userReq := &client.GetUserRequest{UserId: userId}

	return tdlibClient.GetUser(userReq)
}

func GetContent(content client.MessageContent) string {
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
	default:

		return JsonMarshalStr(content)
	}
}

func getChatsList() {
	maxChatId := client.JsonInt64(int64((^uint64(0)) >> 1))
	offsetOrder := maxChatId
	log.Printf("Requesting chats with max id: %d", maxChatId)

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
			offsetChatId = chat.Id
			offsetOrder = chat.Positions[0].Order
		}

		if len(chats.ChatIds) == 0 {
			log.Printf("Reached end of the list")

			break
		}
		time.Sleep(1 * time.Second)
		page++
		log.Println()
	}
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