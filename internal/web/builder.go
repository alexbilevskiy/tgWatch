package web

import (
	"fmt"
	"html"
	"strings"
	"unicode/utf16"

	"github.com/alexbilevskiy/tgWatch/internal/account"
	"github.com/alexbilevskiy/tgWatch/internal/helpers"
	"github.com/alexbilevskiy/tgWatch/internal/tdlib"
	"github.com/zelenin/go-tdlib/client"
)

func parseMessage(acc *account.Account, message *client.Message, verbose bool) MessageInfo {
	senderChatId := tdlib.GetChatIdBySender(message.SenderId)
	ct := GetContentWithText(message.Content, message.ChatId)
	messageInfo := MessageInfo{
		T:             "NewMessage",
		MessageId:     message.Id,
		Date:          message.Date,
		DateTimeStr:   helpers.FormatDateTime(message.Date),
		DateStr:       helpers.FormatDate(message.Date),
		TimeStr:       helpers.FormatTime(message.Date),
		ChatId:        message.ChatId,
		ChatName:      acc.TdApi.GetChatName(message.ChatId),
		SenderId:      senderChatId,
		SenderName:    acc.TdApi.GetSenderName(message.SenderId),
		MediaAlbumId:  int64(message.MediaAlbumId),
		SimpleText:    ct.Text,
		FormattedText: ct.FormattedText,
		Attachments:   GetContentAttachments(message.Content),
		Edited:        message.EditDate != 0,
		ContentRaw:    nil,
	}

	if verbose {
		messageInfo.ContentRaw = message.Content
	}

	return messageInfo
}

func buildChatInfoByLocalChat(acc *account.Account, chat *client.Chat) ChatInfo {
	if chat == nil {

		return ChatInfo{ChatId: -1, Username: "ERROR", ChatName: "NULL CHAT"}
	}
	info := ChatInfo{ChatId: chat.Id, ChatName: acc.TdApi.GetChatName(chat.Id), Username: acc.TdApi.GetChatUsername(chat.Id)}
	switch chat.Type.ChatTypeType() {
	case client.TypeChatTypeSupergroup:
		t := chat.Type.(*client.ChatTypeSupergroup)
		sg, err := acc.TdApi.GetSuperGroup(t.SupergroupId)
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
	case client.TypeChatTypePrivate:
		info.Type = "User"
	case client.TypeChatTypeBasicGroup:
		info.Type = "Group"
	default:
		info.Type = chat.Type.ChatTypeType()
	}
	info.CountUnread = chat.UnreadCount

	return info
}

func RenderText(text *client.FormattedText) string {
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
		if (entity.Offset-prevEntityEnd > 0) || entity.Offset == 0 {
			result += ut2hs(utfText[prevEntityEnd:entity.Offset])
		}
		prevEntityEnd = entity.Offset + entity.Length

		//extract clean string from message
		repl := ut2hs(utfText[entity.Offset : entity.Offset+entity.Length])
		//wrap in html tags according to entity
		wrapped = wrapEntity(entity, repl)

		//if next entity has same offset and length as current, theses entities are nested
		ne := i + 1
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
	case client.TypeTextEntityTypeSpoiler:
		wrapped = fmt.Sprintf(`<span class="spoiler">%s</span>`, text)
	case client.TypeTextEntityTypeCustomEmoji:
		t := entity.Type.(*client.TextEntityTypeCustomEmoji)
		thumbLink := fmt.Sprintf("/e/%d", t.CustomEmojiId)
		wrapped = fmt.Sprintf(`<img width=20 src="%s" alt="%s" title="%s">`, thumbLink, text, text)
		//wrapped = fmt.Sprintf(`<span title="%s" class="badge bg-info">%s</span>`, entity.Type.TextEntityTypeType(), text)
	default:
		wrapped = fmt.Sprintf(`<span title="%s" class="badge bg-danger">%s</span>`, entity.Type.TextEntityTypeType(), text)
	}

	return wrapped
}

func ut2hs(r []uint16) string { //utf text to html string
	return html.EscapeString(string(utf16.Decode(r)))
}

func BuildMessageLink(chatId int64, messageId int64) string {
	return fmt.Sprintf("/m/%d/%d", chatId, messageId)
}
