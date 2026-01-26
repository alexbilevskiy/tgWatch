package utils

import (
	"fmt"
	"html"
	"strings"
	"unicode/utf16"

	"github.com/zelenin/go-tdlib/client"
)

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
	switch entity.Type.TextEntityTypeConstructor() {
	case client.ConstructorTextEntityTypeBold:
		wrapped = "<b>" + text + "</b>"
	case client.ConstructorTextEntityTypeItalic:
		wrapped = "<i>" + text + "</i>"
	case client.ConstructorTextEntityTypeUnderline:
		wrapped = "<u>" + text + "</u>"
	case client.ConstructorTextEntityTypeStrikethrough:
		wrapped = "<s>" + text + "</s>"
	case client.ConstructorTextEntityTypeMention:
		wrapped = fmt.Sprintf(`<a href="https://t.me/%s">%s</a>`, text[1:], text)
	case client.ConstructorTextEntityTypeMentionName:
		t := entity.Type.(*client.TextEntityTypeMentionName)
		wrapped = fmt.Sprintf(`<a href="/h/%d">%s</a>`, t.UserId, text)
	case client.ConstructorTextEntityTypeCode:
		wrapped = "<code>" + text + "</code>"
	case client.ConstructorTextEntityTypeUrl:
		wrapped = fmt.Sprintf(`<a href="%s">%s</a>`, text, text)
	case client.ConstructorTextEntityTypeTextUrl:
		t := entity.Type.(*client.TextEntityTypeTextUrl)
		wrapped = fmt.Sprintf(`<a href="%s">%s</a>`, t.Url, text)
	case client.ConstructorTextEntityTypePre:
		wrapped = "<code>" + text + "</code>"
	case client.ConstructorTextEntityTypeBotCommand:
		wrapped = "<a>" + text + "</a>"
	case client.ConstructorTextEntityTypeHashtag:
		wrapped = "<a>" + text + "</a>"
	case client.ConstructorTextEntityTypeEmailAddress:
		wrapped = fmt.Sprintf(`<a href="mailto:%s">%s</a>`, text, text)
	case client.ConstructorTextEntityTypePhoneNumber:
		wrapped = fmt.Sprintf(`<a href="tel:%s">%s</a>`, text, text)
	case client.ConstructorTextEntityTypeSpoiler:
		wrapped = fmt.Sprintf(`<span class="spoiler">%s</span>`, text)
	case client.ConstructorTextEntityTypeCustomEmoji:
		t := entity.Type.(*client.TextEntityTypeCustomEmoji)
		thumbLink := fmt.Sprintf("/e/%d", t.CustomEmojiId)
		wrapped = fmt.Sprintf(`<img width=20 src="%s" alt="%s" title="%s">`, thumbLink, text, text)
		//wrapped = fmt.Sprintf(`<span title="%s" class="badge bg-info">%s</span>`, entity.Type.TextEntityTypeConstructor(), text)
	default:
		wrapped = fmt.Sprintf(`<span title="%s" class="badge bg-danger">%s</span>`, entity.Type.TextEntityTypeConstructor(), text)
	}

	return wrapped
}

func ut2hs(r []uint16) string { //utf text to html string
	return html.EscapeString(string(utf16.Decode(r)))
}

func BuildMessageLink(chatId int64, messageId int64) string {
	return fmt.Sprintf("/m/%d/%d", chatId, messageId)
}

