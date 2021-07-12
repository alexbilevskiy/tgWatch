package libs

import (
	"fmt"
	"go-tdlib/client"
	"html"
	"strings"
	"tgWatch/structs"
	"unicode/utf16"
)

func parseUpdateNewMessage(upd *client.UpdateNewMessage) structs.MessageInfo {
	senderChatId := GetChatIdBySender(upd.Message.Sender)
	ct := GetContentWithText(upd.Message.Content)
	msg := structs.MessageInfo{
		T:             "NewMessage",
		MessageId:     upd.Message.Id,
		Date:          upd.Message.Date,
		DateTimeStr:   FormatDateTime(upd.Message.Date),
		DateStr:       FormatDate(upd.Message.Date),
		TimeStr:       FormatTime(upd.Message.Date),
		ChatId:        upd.Message.ChatId,
		ChatName:      GetChatName(upd.Message.ChatId),
		SenderId:      senderChatId,
		SenderName:    GetSenderName(upd.Message.Sender),
		MediaAlbumId:  int64(upd.Message.MediaAlbumId),
		SimpleText:    ct.Text,
		FormattedText: ct.FormattedText,
		Attachments:   GetContentAttachments(upd.Message.Content),
		Deleted:       IsMessageDeleted(upd.Message.ChatId, upd.Message.Id),
		Edited:        IsMessageEdited(upd.Message.ChatId, upd.Message.Id),
		ContentRaw:    nil,
	}

	if verbose {
		msg.ContentRaw = upd.Message.Content
	}

	return msg
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

func renderText(text *client.FormattedText) string {
	utfText := utf16.Encode([]rune(text.Text))
	res := ""
	var prevOffset int32 = 0

	for _, entity := range text.Entities {
		if (entity.Offset - prevOffset > 0) || entity.Offset == 0 {
			res += ut2hs(utfText[prevOffset:entity.Offset])
		}
		prevOffset = entity.Offset + entity.Length
		if int32(len(utfText)) < entity.Offset + entity.Length {
			res += "ERROR!"
			break
		}
		repl := ut2hs(utfText[entity.Offset:entity.Offset + entity.Length])

		switch entity.Type.TextEntityTypeType() {
		case client.TypeTextEntityTypeBold:
			res += "<b>" + repl + "</b>"
		case client.TypeTextEntityTypeItalic:
			res += "<i>" + repl + "</i>"
		case client.TypeTextEntityTypeUnderline:
			res += "<u>" + repl + "</u>"
		case client.TypeTextEntityTypeStrikethrough:
			res += "<s>" + repl + "</s>"
		case client.TypeTextEntityTypeMention:
			res += fmt.Sprintf(`<a href="https://t.me/%s">%s</a>`, repl[1:], repl)
		case client.TypeTextEntityTypeMentionName:
			t := entity.Type.(*client.TextEntityTypeMentionName)
			res += fmt.Sprintf(`<a href="/h/%d">%s</a>`, t.UserId, repl)
		case client.TypeTextEntityTypeCode:
			res += "<code>" + repl + "</code>"
		case client.TypeTextEntityTypeUrl:
			res += fmt.Sprintf(`<a href="%s">%s</a>`, repl, repl)
		case client.TypeTextEntityTypeTextUrl:
			t := entity.Type.(*client.TextEntityTypeTextUrl)
			res += fmt.Sprintf(`<a href="%s">%s</a>`, t.Url, repl)
		case client.TypeTextEntityTypePre:
			res += "<code>" + repl + "</code>"
		case client.TypeTextEntityTypeBotCommand:
			res += "<a>" + repl + "</a>"
		case client.TypeTextEntityTypeHashtag:
			res += "<a>" + repl + "</a>"
		case client.TypeTextEntityTypeEmailAddress:
			res += fmt.Sprintf(`<a href="mailto:%s">%s</a>`, repl, repl)
		default:
			res += fmt.Sprintf(`<span title="%s" class="badge bg-danger">%s</span>`, entity.Type.TextEntityTypeType(), repl)
		}
	}
	if int32(len(utfText)) > prevOffset {
		res += ut2hs(utfText[prevOffset:])
	}
	res = strings.Replace(res, "\n", "<br>", -1)

	return res
}

func ut2hs(r []uint16) string { //utf text to html string
	return html.EscapeString(string(utf16.Decode(r)))
}
