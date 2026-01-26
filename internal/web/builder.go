package web

import (
	"context"

	"github.com/alexbilevskiy/tgwatch/internal/account"
	"github.com/alexbilevskiy/tgwatch/internal/helpers"
	"github.com/alexbilevskiy/tgwatch/internal/tdlib"
	"github.com/alexbilevskiy/tgwatch/internal/web/models"
	"github.com/alexbilevskiy/tgwatch/internal/web/utils"
	"github.com/zelenin/go-tdlib/client"
)

func parseMessage(ctx context.Context, acc *account.Account, message *client.Message, verbose bool) models.MessageInfo {
	senderChatId := tdlib.GetChatIdBySender(message.SenderId)
	ct := utils.GetContentWithText(message.Content, message.ChatId)
	messageInfo := models.MessageInfo{
		T:             "NewMessage",
		MessageId:     message.Id,
		Date:          message.Date,
		DateTimeStr:   helpers.FormatDateTime(message.Date),
		DateStr:       helpers.FormatDate(message.Date),
		TimeStr:       helpers.FormatTime(message.Date),
		ChatId:        message.ChatId,
		ChatName:      acc.TdApi.GetChatName(ctx, message.ChatId),
		SenderId:      senderChatId,
		SenderName:    acc.TdApi.GetSenderName(ctx, message.SenderId),
		MediaAlbumId:  int64(message.MediaAlbumId),
		SimpleText:    ct.Text,
		FormattedText: ct.FormattedText,
		Attachments:   utils.GetContentAttachments(message.Content),
		Edited:        message.EditDate != 0,
		ContentRaw:    nil,
	}

	if verbose {
		messageInfo.ContentRaw = message.Content
	}

	return messageInfo
}

func buildChatInfoByLocalChat(ctx context.Context, acc *account.Account, chat *client.Chat) models.ChatInfo {
	if chat == nil {

		return models.ChatInfo{ChatId: -1, Username: "ERROR", ChatName: "NULL CHAT"}
	}
	info := models.ChatInfo{ChatId: chat.Id, ChatName: acc.TdApi.GetChatName(ctx, chat.Id), Username: acc.TdApi.GetChatUsername(ctx, chat.Id)}
	switch chat.Type.ChatTypeConstructor() {
	case client.ConstructorChatTypeSupergroup:
		t := chat.Type.(*client.ChatTypeSupergroup)
		sg, err := acc.TdApi.GetSuperGroup(ctx, t.SupergroupId)
		if err != nil {
			info.Type = "Error " + err.Error()
		} else {
			if sg.IsChannel {
				info.Type = "Channel"
			} else {
				info.Type = "Supergroup"
			}
			info.HasTopics = sg.IsForum
		}
	case client.ConstructorChatTypePrivate:
		info.Type = "User"
	case client.ConstructorChatTypeBasicGroup:
		info.Type = "Group"
	default:
		info.Type = chat.Type.ChatTypeConstructor()
	}
	info.CountUnread = chat.UnreadCount

	return info
}
