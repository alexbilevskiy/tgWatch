package main

import (
	"context"
	"errors"
	"fmt"
	"go-tdlib/client"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"path/filepath"
	"strconv"
	"tgWatch/config"
	"tgWatch/structs"
	"time"
)

var tdlibClient *client.Client
var mongoClient *mongo.Client
var mongoContext context.Context
var updatesColl *mongo.Collection

func main() {
	initLibs()

	me, err := tdlibClient.GetMe()
	if err != nil {
		log.Fatalf("GetMe error: %s", err)
	}

	log.Printf("Me: %s %s [%s]", me.FirstName, me.LastName, me.Username)

	listenUpdates()
}

func initLibs() {
	config.InitConfiguration()
	mongoContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	mongoClient, err := mongo.Connect(mongoContext, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatalf("Mongo error: %s", err)
	}
	updatesColl = mongoClient.Database("tg").Collection("updates")

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
}

func listenUpdates()  {
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
				mongoId := saveUpdate(t, upd, upd.Message.Date)

				req := &client.GetChatRequest{ChatId: upd.Message.ChatId}
				fullChat, err := tdlibClient.GetChat(req)
				if err != nil {
					log.Printf("[%s] New Message from chat: %d, ERROR %s", mongoId, upd.Message.ChatId, err)
				} else {
					log.Printf("[%s] New Message from chat: %d, `%s`", mongoId, upd.Message.ChatId, fullChat.Title)
				}

				break
			case "updateMessageEdited":
				upd := update.(*client.UpdateMessageEdited)
				if config.Config.IgnoreChatIds[strconv.FormatInt(upd.ChatId, 10)] {

					break
				}
				mongoId := saveUpdate(t, upd, upd.EditDate)
				req := &client.GetChatRequest{ChatId: upd.ChatId}
				fullChat, err := tdlibClient.GetChat(req)
				if err != nil {
					log.Printf("[%s] EDITED msg! Chat: %d, msg %d, ERROR %s", mongoId, upd.ChatId, upd.MessageId, err)
				} else {
					log.Printf("[%s] EDITED msg! Chat: %d, msg %d, `%s`", mongoId, upd.ChatId, upd.MessageId, fullChat.Title)
				}

				break
			case "updateMessageContent":
				upd := update.(*client.UpdateMessageContent)
				if config.Config.IgnoreChatIds[strconv.FormatInt(upd.ChatId, 10)] {

					break
				}
				mongoId := saveUpdate(t, upd, 0)
				req := &client.GetChatRequest{ChatId: upd.ChatId}
				fullChat, err := tdlibClient.GetChat(req)
				if err != nil {
					log.Printf("[%s] EDITED content! Chat: %d, msg %d, ERROR %s", mongoId, upd.ChatId, upd.MessageId, err)
				} else {
					log.Printf("[%s] EDITED content! Chat: %d, msg %d, `%s`", mongoId, upd.ChatId, upd.MessageId, fullChat.Title)
				}

				break
			default:
				log.Printf("%s : %#v", t, update)
			}
		}
	}
}

func getContent(content client.MessageContent) (string, error) {
	cType := content.MessageContentType()
	switch cType {
		case "messageText":
			msg := content.(*client.MessageText)
			return fmt.Sprintf("%s", msg.Text.Text), nil
	default:
		return "", errors.New(fmt.Sprintf("Message type %s not supported", cType));
	}
}

func saveUpdate(t string, upd interface{}, timestamp int32) string {
	if timestamp == 0 {
		timestamp = int32(time.Now().Unix())
	}
	update := structs.TgUpdate{T: t, Time: timestamp, Upd: upd}
	res, err := updatesColl.InsertOne(mongoContext, update)
	if err != nil {
		log.Printf("[ERROR] insert %s: %s", t, err)
		return ""
	}
	return res.InsertedID.(primitive.ObjectID).String()
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
