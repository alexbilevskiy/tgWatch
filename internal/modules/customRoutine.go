package modules

import (
	"context"
	"log"
	"strings"

	"github.com/zelenin/go-tdlib/client"
)

// @TODO: create some kind of lua integration to allow writing custom message processing plugins without need to recompile

var dudeChatId int64 = -1002150910059
var repostMsgId int64 = 6410335158272
var myUserId int64 = 118137353
var tgChatId int64 = 777000
var myUsername string = "alexbilevskiy"

func sendCoffee(ctx context.Context, tdlibClient *client.Client, content client.MessageContent) {
	if content.MessageContentConstructor() != client.ConstructorMessageText {
		return
	}
	cnt := content.(*client.MessageText)
	if !strings.Contains(strings.ToLower(cnt.Text.Text), "по кофейку!") {
		return
	}
	log.Printf("Sending coffee!!!")
	req := &client.ForwardMessagesRequest{ChatId: dudeChatId, FromChatId: myUserId, MessageIds: append(make([]int64, 0), repostMsgId), RemoveCaption: true, SendCopy: true}
	messages, err := tdlibClient.ForwardMessages(ctx, req)
	if err != nil {
		log.Printf("Failed to send coffee: %s", err.Error())
	} else {
		log.Printf("Sent coffee! count: %d", messages.TotalCount)
	}
}

func sendTgNotification(ctx context.Context, acc int64, tdlibClient *client.Client, update *client.UpdateNewMessage) {
	gcReq := client.GetChatRequest{ChatId: myUserId}
	_, err := tdlibClient.GetChat(ctx, &gcReq)
	if err != nil {
		log.Printf("Failed to get chat (%s), trying to create", err.Error())

		srReq := client.SearchPublicChatRequest{Username: myUsername}
		_, err := tdlibClient.SearchPublicChat(ctx, &srReq)
		if err != nil {
			log.Printf("Failed to search public chat: %s", err.Error())
			return
		}
		chReq := client.CreatePrivateChatRequest{UserId: myUserId}
		_, err = tdlibClient.CreatePrivateChat(ctx, &chReq)
		if err != nil {
			log.Printf("Failed to create private chat: %s", err.Error())
			return
		}
	}
	req := client.SendMessageRequest{ChatId: myUserId, InputMessageContent: &client.InputMessageText{Text: &client.FormattedText{Text: "got new message from tg"}}}
	_, err = tdlibClient.SendMessage(ctx, &req)
	if err != nil {
		log.Printf("Failed to notify: %s", err.Error())
		return
	}
	log.Printf("[%d] New notification from tg: %d", acc, update.Message.Id)

}

func CustomNewMessageRoutine(ctx context.Context, acc int64, tdlibClient *client.Client, update *client.UpdateNewMessage) {
	if acc != myUserId {
		if update.Message.ChatId == tgChatId {
			sendTgNotification(ctx, acc, tdlibClient, update)
		}

		return
	}

	if update.Message.ChatId == dudeChatId {
		sendCoffee(ctx, tdlibClient, update.Message.Content)

		return
	}

	return
}

func CustomMessageContentRoutine(ctx context.Context, acc int64, tdlibClient *client.Client, update *client.UpdateMessageContent) {
	if acc != myUserId {
		return
	}

	if update.ChatId == dudeChatId {
		sendCoffee(ctx, tdlibClient, update.NewContent)

		return
	}
}
