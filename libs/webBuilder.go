package libs

import (
	"fmt"
	"go-tdlib/client"
	"tgWatch/structs"
)

func parseUpdateMessageEdited(upd *client.UpdateMessageEdited) structs.MessageEditedMeta {
	m := structs.MessageEditedMeta{
		T:         "EditedMeta",
		MessageId: upd.MessageId,
		Date:      upd.EditDate,
		DateStr:   FormatTime(upd.EditDate),
	}

	return m
}

func parseUpdateNewMessage(upd *client.UpdateNewMessage) structs.MessageInfo {
	content := GetContent(upd.Message.Content)

	senderChatId := GetChatIdBySender(upd.Message.Sender)

	result := structs.MessageInfo{
		T:            "NewMessage",
		MessageId:    upd.Message.Id,
		Date:         upd.Message.Date,
		DateStr:      FormatTime(upd.Message.Date),
		ChatId:       upd.Message.ChatId,
		ChatName:     GetChatName(upd.Message.ChatId),
		SenderId:     senderChatId,
		SenderName:   GetSenderName(upd.Message.Sender),
		Content:      content,
		Attachments:  GetContentStructs(upd.Message.Content),
		ContentRaw:   nil,
		MediaAlbumId: int64(upd.Message.MediaAlbumId),
	}
	if verbose {
		result.ContentRaw = upd.Message.Content
	}

	return result
}

func parseUpdateMessageContent(upd *client.UpdateMessageContent) structs.MessageNewContent {
	result := structs.MessageNewContent{
		T:          "NewContent",
		MessageId:  upd.MessageId,
		Content:    GetContent(upd.NewContent),
		ContentRaw: nil,
	}
	if verbose {
		result.ContentRaw = upd.NewContent
	}

	return result
}

func parseUpdateDeleteMessages(upd *client.UpdateDeleteMessages, date int32) structs.DeleteMessages {
	result := structs.DeleteMessages{
		T:          "DeleteMessages",
		MessageIds: upd.MessageIds,
		ChatId:     upd.ChatId,
		ChatName:   GetChatName(upd.ChatId),
		Date:       date,
		DateStr:    FormatTime(date),
	}
	for _, messageId := range upd.MessageIds {
		m, err := FindUpdateNewMessage(upd.ChatId, messageId)
		if err != nil {
			result.Messages = append(result.Messages, structs.MessageError{T: "Error", MessageId: messageId, Error: fmt.Sprintf("not found deleted message %s", err)})
			continue
		}
		result.Messages = append(result.Messages, parseUpdateNewMessage(m))
	}

	return result
}

