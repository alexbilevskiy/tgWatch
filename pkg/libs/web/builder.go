package web

import (
	"fmt"
	"github.com/alexbilevskiy/tgWatch/pkg/config"
	"github.com/alexbilevskiy/tgWatch/pkg/libs"
	"github.com/alexbilevskiy/tgWatch/pkg/libs/tdlib"
	"github.com/alexbilevskiy/tgWatch/pkg/structs"
	"github.com/zelenin/go-tdlib/client"
	"html"
	"strings"
	"unicode/utf16"
)

func parseMessage(message *client.Message) structs.MessageInfo {
	senderChatId := tdlib.GetChatIdBySender(message.SenderId)
	ct := tdlib.GetContentWithText(message.Content, message.ChatId)
	messageInfo := structs.MessageInfo{
		T:             "NewMessage",
		MessageId:     message.Id,
		Date:          message.Date,
		DateTimeStr:   libs.FormatDateTime(message.Date),
		DateStr:       libs.FormatDate(message.Date),
		TimeStr:       libs.FormatTime(message.Date),
		ChatId:        message.ChatId,
		ChatName:      libs.AS.Get(currentAcc).TdApi.GetChatName(message.ChatId),
		SenderId:      senderChatId,
		SenderName:    libs.AS.Get(currentAcc).TdApi.GetSenderName(message.SenderId),
		MediaAlbumId:  int64(message.MediaAlbumId),
		SimpleText:    ct.Text,
		FormattedText: ct.FormattedText,
		Attachments:   tdlib.GetContentAttachments(message.Content),
		Edited:        message.EditDate != 0,
		ContentRaw:    nil,
	}

	if verbose {
		messageInfo.ContentRaw = message.Content
	}

	return messageInfo
}

func buildChatInfoByLocalChat(chat *client.Chat) structs.ChatInfo {
	if chat == nil {

		return structs.ChatInfo{ChatId: -1, Username: "ERROR", ChatName: "NULL CHAT"}
	}
	info := structs.ChatInfo{ChatId: chat.Id, ChatName: libs.AS.Get(currentAcc).TdApi.GetChatName(chat.Id), Username: libs.AS.Get(currentAcc).TdApi.GetChatUsername(chat.Id)}
	switch chat.Type.ChatTypeType() {
	case client.TypeChatTypeSupergroup:
		t := chat.Type.(*client.ChatTypeSupergroup)
		sg, err := libs.AS.Get(currentAcc).TdApi.GetSuperGroup(t.SupergroupId)
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
		customEmojisIds := append(make([]client.JsonInt64, 1), t.CustomEmojiId)
		customEmojis, err := libs.AS.Get(currentAcc).TdApi.GetCustomEmoji(customEmojisIds)
		if err != nil {
			wrapped = fmt.Sprintf(`<span title="%s" class="badge bg-warning">%s</span>`, entity.Type.TextEntityTypeType(), text)
			break
		}
		if customEmojis.Stickers[0].Thumbnail != nil {
			thumbLink := fmt.Sprintf("/f/%s", customEmojis.Stickers[0].Thumbnail.File.Remote.Id)
			wrapped = fmt.Sprintf(`<img width=20 src="%s" alt="%s" title="%s">`, thumbLink, text, text)
			break
		}
		wrapped = fmt.Sprintf(`<span title="%s" class="badge bg-info">%s</span>`, entity.Type.TextEntityTypeType(), text)

	default:
		wrapped = fmt.Sprintf(`<span title="%s" class="badge bg-danger">%s</span>`, entity.Type.TextEntityTypeType(), text)
	}

	return wrapped
}

func ut2hs(r []uint16) string { //utf text to html string
	return html.EscapeString(string(utf16.Decode(r)))
}

func BuildMessageLink(chatId int64, messageId int64) string {
	return fmt.Sprintf("http://%s/m/%d/%d", config.Config.WebListen, chatId, messageId)
}
