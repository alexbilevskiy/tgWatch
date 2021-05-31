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
		DateStr:   FormatDateTime(upd.EditDate),
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
		DateStr:      FormatDateTime(upd.Message.Date),
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
		DateStr:    FormatDateTime(date),
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

func buildChatInfoByLocalChat(chat *client.Chat, buildCounters bool) structs.ChatInfo {
	info := structs.ChatInfo{ChatId: chat.Id, ChatName: GetChatName(chat.Id)}
	switch chat.Type.ChatTypeType() {
	case client.TypeChatTypeSupergroup:
		t := chat.Type.(*client.ChatTypeSupergroup)
		sg, err := GetSuperGroup(t.SupergroupId)
		if err != nil {
			info.Username = "Error " + err.Error()
		} else {
			if sg.IsChannel {
				info.Type = "Channel"
			} else {
				info.Type = "Supergroup"
			}
			if sg.Username != "" {
				info.Username = sg.Username
			}
		}
	case client.TypeChatTypePrivate:
		t := chat.Type.(*client.ChatTypePrivate)
		info.Type = "User"
		user, err := GetUser(t.UserId)
		if err != nil {
			info.Username = "Error " + err.Error()
		} else {
			if user.Username != "" {
				info.Username = user.Username
			}
		}
	case client.TypeChatTypeBasicGroup:
		//t := chat.Type.(*client.ChatTypeBasicGroup)
		info.Type = "Group"
	default:
		info.Type = chat.Type.ChatTypeType()
	}
	if buildCounters {
		chatStats, err := GetChatsStats(append(make([]int64, 0), chat.Id))
		if err != nil {
			fmt.Printf("Failed to get chat stats %d", chat.Id)
		} else if len(chatStats) > 0 {
			info.CountTotal = chatStats[0].Counters["total"]
			info.CountDeletes = chatStats[0].Counters["updateDeleteMessages"]
			info.CountEdits = chatStats[0].Counters["updateMessageEdited"]
			info.CountMessages = chatStats[0].Counters["updateNewMessage"]
		}
	}

	return info
}
