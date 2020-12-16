package helpers

import (
	"encoding/json"
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
			case "updateChatFilters":

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

			case "updateNewChat":
			case "updateHavePendingNotifications":
			case "updateSupergroupFullInfo":
			case "updateSupergroup":
			case "updateBasicGroup":
			case "updateBasicGroupFullInfo":
			case "updateChatPhoto":
			case "updateUser":
			case "updateChatTitle":
			case "updateDeleteMessages":
				break

			case "updateNewMessage":
				upd := update.(*client.UpdateNewMessage)
				senderChatId := int64(0)
				if upd.Message.Sender.MessageSenderType() == "messageSenderChat" {
					senderChatId = upd.Message.Sender.(*client.MessageSenderChat).ChatId
				} else if upd.Message.Sender.MessageSenderType() == "messageSenderUser" {
					senderChatId = int64(upd.Message.Sender.(*client.MessageSenderUser).UserId)
				}
				if config.Config.IgnoreChatIds[strconv.FormatInt(upd.Message.ChatId, 10)] || config.Config.IgnoreAuthorIds[strconv.FormatInt(senderChatId, 10)] {

					break
				}
				mongoId := SaveUpdate(t, upd, upd.Message.Date)

				link := GetLink(tdlibClient, upd.Message.ChatId, upd.Message.Id)
				chatName := GetChatName(tdlibClient, upd.Message.ChatId)
				intLink := fmt.Sprintf("http://%s/%d/%d", config.Config.WebListen, upd.Message.ChatId, upd.Message.Id)
				log.Printf("[%s] New Message from chat: %d, %s, %s, %s", mongoId, upd.Message.ChatId, chatName, link, intLink)

				break
			case "updateMessageEdited":
				upd := update.(*client.UpdateMessageEdited)
				if config.Config.IgnoreChatIds[strconv.FormatInt(upd.ChatId, 10)] {

					break
				}

				if upd.ReplyMarkup != nil {
					//log.Printf("SKIP EDITED msg! Chat: %d, msg %d, %s | %s", upd.ChatId, upd.MessageId, chatName, jsonMarshalStr(upd.ReplyMarkup))

					break
				}
				mongoId := SaveUpdate(t, upd, upd.EditDate)
				link := GetLink(tdlibClient, upd.ChatId, upd.MessageId)
				chatName := GetChatName(tdlibClient, upd.ChatId)
				log.Printf("[%s] EDITED msg! Chat: %d, msg %d, %s, %s", mongoId, upd.ChatId, upd.MessageId, chatName, link)

				break
			case "updateMessageContent":
				upd := update.(*client.UpdateMessageContent)
				if config.Config.IgnoreChatIds[strconv.FormatInt(upd.ChatId, 10)] {

					break
				}
				if upd.NewContent.MessageContentType() == "messagePoll" {
					//dont save "poll" updates - that's just counters, users cannot update polls manually
					break
				}
				mongoId := SaveUpdate(t, upd, 0)

				link := GetLink(tdlibClient, upd.ChatId, upd.MessageId)
				chatName := GetChatName(tdlibClient, upd.ChatId)
				log.Printf("[%s] EDITED content! Chat: %d, msg %d, %s, %s", mongoId, upd.ChatId, upd.MessageId, chatName, link)
				log.Printf("%s", GetContent(upd.NewContent))

				break
			default:
				log.Printf("%s : %#v", t, update)
			}
		}
	}
}

func GetLink(tdlibClient *client.Client, chatId int64, messageId int64) string {
	linkReq := &client.GetMessageLinkRequest{ChatId: chatId, MessageId: messageId}
	link, err := tdlibClient.GetMessageLink(linkReq)
	if err != nil {

		return "no_link"
	}

	return link.Link
}

func GetChatName(tdlibClient *client.Client, chatId int64) string {
	req := &client.GetChatRequest{ChatId: chatId}
	fullChat, err := tdlibClient.GetChat(req)
	if err != nil {

		return "no_title"
	}

	return fmt.Sprintf("`%s`", fullChat.Title)
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

		return fmt.Sprintf("Photo, %s", msg.Caption.Text)
	case "messagePoll":
		msg := content.(*client.MessagePoll)

		return fmt.Sprintf("Poll, %s", msg.Poll.Question)
	default:

		return jsonMarshalStr(content)
	}
}

func jsonMarshalStr(j interface{}) string {
	m, err := json.Marshal(j)
	if err != nil {

		return "INVALID_JSON"
	}

	return string(m)
}


func getChatsList(tdlibClient *client.Client, ) {
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