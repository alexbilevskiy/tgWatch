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
	ct := GetContentWithText(upd.Message.Content, upd.Message.ChatId)
	msg := structs.MessageInfo{
		T:             "NewMessage",
		MessageId:     upd.Message.Id,
		Date:          upd.Message.Date,
		DateTimeStr:   FormatDateTime(upd.Message.Date),
		DateStr:       FormatDate(upd.Message.Date),
		TimeStr:       FormatTime(upd.Message.Date),
		ChatId:        upd.Message.ChatId,
		ChatName:      GetChatName(currentAcc, upd.Message.ChatId),
		SenderId:      senderChatId,
		SenderName:    GetSenderName(currentAcc, upd.Message.Sender),
		MediaAlbumId:  int64(upd.Message.MediaAlbumId),
		SimpleText:    ct.Text,
		FormattedText: ct.FormattedText,
		Attachments:   GetContentAttachments(upd.Message.Content),
		Deleted:       IsMessageDeleted(currentAcc, upd.Message.ChatId, upd.Message.Id),
		Edited:        IsMessageEdited(currentAcc, upd.Message.ChatId, upd.Message.Id),
		ContentRaw:    nil,
	}

	if verbose {
		msg.ContentRaw = upd.Message.Content
	}

	return msg
}

func buildChatInfoByLocalChat(chat *client.Chat, buildCounters bool) structs.ChatInfo {
	if chat == nil {

		return structs.ChatInfo{ChatId: -1, Username: "ERROR", ChatName: "NULL CHAT"}
	}
	info := structs.ChatInfo{ChatId: chat.Id, ChatName: GetChatName(currentAcc, chat.Id)}
	switch chat.Type.ChatTypeType() {
	case client.TypeChatTypeSupergroup:
		t := chat.Type.(*client.ChatTypeSupergroup)
		sg, err := GetSuperGroup(currentAcc, t.SupergroupId)
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
		user, err := GetUser(currentAcc, t.UserId)
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
		chatStats, err := GetChatsStats(currentAcc, append(make([]int64, 0), chat.Id))
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
	result := ""
	var prevEntityEnd int32 = 0

	var wrapped string
	for i, entity := range text.Entities {
		if entity.Offset < prevEntityEnd {
			//fmt.Printf("ENT IS SAME AS PREV\n")
			wrapped = wrapEntity(entity, wrapped)
			result += wrapped

			continue
		}
		//check if there is plain text between current entity and previous
		if (entity.Offset - prevEntityEnd > 0) || entity.Offset == 0 {
			result += ut2hs(utfText[prevEntityEnd:entity.Offset])
		}
		prevEntityEnd = entity.Offset + entity.Length

		//extract clean string from message
		repl := ut2hs(utfText[entity.Offset:entity.Offset + entity.Length])
		//wrap in html tags according to entity
		wrapped = wrapEntity(entity, repl)

		//if next entity has same offset and length as current, theses entities are nested
		ne := i+1
		if len(text.Entities) > ne && entity.Offset == text.Entities[ne].Offset && entity.Length == text.Entities[ne].Length {
			//fmt.Printf("NEXT IS SAME AS CUR\n")
			continue
		}
		result += wrapped
	}

	//check if there is plain text after last entity
	if int32(len(utfText)) > prevEntityEnd {
		result += ut2hs(utfText[prevEntityEnd:])
	}
	result = strings.Replace(result, "\n", "<br>", -1)

	return result
}

func wrapEntity(entity *client.TextEntity, text string) string {
	var wrapped string
	switch entity.Type.TextEntityTypeType() {
	case client.TypeTextEntityTypeBold:
		wrapped = "<b>" + text + "</b>"
	case client.TypeTextEntityTypeItalic:
		wrapped = "<i>" + text + "</i>"
	case client.TypeTextEntityTypeUnderline:
		wrapped = "<u>" + text + "</u>"
	case client.TypeTextEntityTypeStrikethrough:
		wrapped = "<s>" + text + "</s>"
	case client.TypeTextEntityTypeMention:
		wrapped = fmt.Sprintf(`<a href="https://t.me/%s">%s</a>`, text[1:], text)
	case client.TypeTextEntityTypeMentionName:
		t := entity.Type.(*client.TextEntityTypeMentionName)
		wrapped = fmt.Sprintf(`<a href="/h/%d">%s</a>`, t.UserId, text)
	case client.TypeTextEntityTypeCode:
		wrapped = "<code>" + text + "</code>"
	case client.TypeTextEntityTypeUrl:
		wrapped = fmt.Sprintf(`<a href="%s">%s</a>`, text, text)
	case client.TypeTextEntityTypeTextUrl:
		t := entity.Type.(*client.TextEntityTypeTextUrl)
		wrapped = fmt.Sprintf(`<a href="%s">%s</a>`, t.Url, text)
	case client.TypeTextEntityTypePre:
		wrapped = "<code>" + text + "</code>"
	case client.TypeTextEntityTypeBotCommand:
		wrapped = "<a>" + text + "</a>"
	case client.TypeTextEntityTypeHashtag:
		wrapped = "<a>" + text + "</a>"
	case client.TypeTextEntityTypeEmailAddress:
		wrapped = fmt.Sprintf(`<a href="mailto:%s">%s</a>`, text, text)
	case client.TypeTextEntityTypePhoneNumber:
		wrapped = fmt.Sprintf(`<a href="tel:%s">%s</a>`, text, text)
	default:
		wrapped = fmt.Sprintf(`<span title="%s" class="badge bg-danger">%s</span>`, entity.Type.TextEntityTypeType(), text)
	}

	return wrapped
}

func ut2hs(r []uint16) string { //utf text to html string
	return html.EscapeString(string(utf16.Decode(r)))
}
