package modules

import (
	"github.com/zelenin/go-tdlib/client"
	"log"
	"strings"
)

// @TODO: create some kind of lua integration to allow writing custom message processing plugins without need to recompile

var dudeChatId int64 = -1002150910059
var repostMsgId int64 = 6410335158272
var myUserId int64 = 118137353

func sendCoffee(tdlibClient *client.Client, content client.MessageContent) {
	if content.MessageContentType() != client.TypeMessageText {
		return
	}
	cnt := content.(*client.MessageText)
	if !strings.Contains(strings.ToLower(cnt.Text.Text), "по кофейку!") {
		return
	}
	log.Printf("Sending coffee!!!")
	req := &client.ForwardMessagesRequest{ChatId: dudeChatId, FromChatId: myUserId, MessageIds: append(make([]int64, 0), repostMsgId), RemoveCaption: true, SendCopy: true}
	messages, err := tdlibClient.ForwardMessages(req)
	if err != nil {
		log.Printf("Failed to send coffee: %s", err.Error())
	} else {
		log.Printf("Sent coffee! count: %d", messages.TotalCount)
	}
}

func CustomNewMessageRoutine(acc int64, tdlibClient *client.Client, update *client.UpdateNewMessage) {
	if acc != myUserId {
		return
	}

	if update.Message.ChatId == dudeChatId {
		sendCoffee(tdlibClient, update.Message.Content)

		return
	}

	return
}

func CustomMessageContentRoutine(acc int64, tdlibClient *client.Client, update *client.UpdateMessageContent) {
	if acc != myUserId {
		return
	}

	if update.ChatId == dudeChatId {
		sendCoffee(tdlibClient, update.NewContent)

		return
	}
}
