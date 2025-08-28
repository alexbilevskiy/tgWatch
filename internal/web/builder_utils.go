package web

import (
	"encoding/base64"
	"fmt"
	"log"

	"github.com/alexbilevskiy/tgWatch/internal/helpers"
	"github.com/zelenin/go-tdlib/client"
)

func GetContentWithText(content client.MessageContent, chatId int64) MessageTextContent {
	if content == nil {

		return MessageTextContent{Text: "UNSUPPORTED_CONTENT"}
	}

	cType := content.MessageContentConstructor()
	switch cType {
	case client.ConstructorMessageText:
		msg := content.(*client.MessageText)

		return MessageTextContent{FormattedText: msg.Text}
	case client.ConstructorMessagePhoto:
		msg := content.(*client.MessagePhoto)

		return MessageTextContent{FormattedText: msg.Caption}
	case client.ConstructorMessageVideo:
		msg := content.(*client.MessageVideo)

		return MessageTextContent{FormattedText: msg.Caption}
	case client.ConstructorMessageAnimation:
		msg := content.(*client.MessageAnimation)

		return MessageTextContent{FormattedText: msg.Caption}
	case client.ConstructorMessagePoll:
		msg := content.(*client.MessagePoll)

		return MessageTextContent{Text: fmt.Sprintf("Poll, %s", msg.Poll.Question)}
	case client.ConstructorMessageSticker:
		msg := content.(*client.MessageSticker)

		return MessageTextContent{Text: fmt.Sprintf("%s sticker", msg.Sticker.Emoji)}
	case client.ConstructorMessageVoiceNote:
		msg := content.(*client.MessageVoiceNote)

		return MessageTextContent{FormattedText: msg.Caption}
	case client.ConstructorMessageVideoNote:
		return MessageTextContent{Text: ""}
	case client.ConstructorMessageDocument:
		msg := content.(*client.MessageDocument)

		return MessageTextContent{FormattedText: msg.Caption}
	case client.ConstructorMessageChatAddMembers:
		msg := content.(*client.MessageChatAddMembers)

		return MessageTextContent{Text: fmt.Sprintf("Added users %s", helpers.JsonMarshalStr(msg.MemberUserIds))}
	case client.ConstructorMessagePinMessage:
		msg := content.(*client.MessagePinMessage)
		var url client.TextEntityType
		//@TODO: where to get chat ID?
		url = &client.TextEntityTypeTextUrl{Url: fmt.Sprintf("/m/%d/%d", chatId, msg.MessageId)}
		entity := &client.TextEntity{Type: url, Offset: 0, Length: 6}
		t := &client.FormattedText{Text: "Pinned message", Entities: append(make([]*client.TextEntity, 0), entity)}

		return MessageTextContent{FormattedText: t}
	case client.ConstructorMessageCall:
		msg := content.(*client.MessageCall)

		return MessageTextContent{Text: fmt.Sprintf("Call (%ds)", msg.Duration)}
	case client.ConstructorMessageAnimatedEmoji:
		msg := content.(*client.MessageAnimatedEmoji)
		if msg.AnimatedEmoji.Sticker != nil {

			return MessageTextContent{Text: fmt.Sprintf("%s (animated)", msg.AnimatedEmoji.Sticker.Emoji)}
		}
		return MessageTextContent{Text: "(invalid animated sticker)"}

	case client.ConstructorMessageChatChangeTitle:
		msg := content.(*client.MessageChatChangeTitle)

		return MessageTextContent{Text: fmt.Sprintf("Chat name was changed to '%s'", msg.Title)}
	case client.ConstructorMessageScreenshotTaken:

		return MessageTextContent{Text: "has taken screenshot!"}
	case client.ConstructorMessageChatJoinByLink:

		return MessageTextContent{Text: "joined by invite link"}
	case client.ConstructorMessageChatDeleteMember:
		msg := content.(*client.MessageChatDeleteMember)
		//@TODO: pass currentAcc as argument
		return MessageTextContent{Text: fmt.Sprintf("deleted `%d` from chat", msg.UserId)}
	case client.ConstructorMessageUnsupported:
		//msg := content.(*client.MessageUnsupported)
		return MessageTextContent{Text: ">unsupported message<"}
	default:
		log.Printf("unknown text type: %s", content.MessageContentConstructor())

		return MessageTextContent{Text: helpers.JsonMarshalStr(content)}
	}
}

func GetContentAttachments(content client.MessageContent) []MessageAttachment {
	if content == nil {

		return nil
	}
	cType := content.MessageContentConstructor()
	var cnt []MessageAttachment
	switch cType {
	case client.ConstructorMessagePhoto:
		msg := content.(*client.MessagePhoto)
		s := MessageAttachment{
			T:  msg.Photo.GetConstructor(),
			Id: msg.Photo.Sizes[len(msg.Photo.Sizes)-1].Photo.Remote.Id,
		}
		if msg.Photo.Minithumbnail != nil {
			s.Thumb = base64.StdEncoding.EncodeToString(msg.Photo.Minithumbnail.Data)
		}
		for _, size := range msg.Photo.Sizes {
			s.Link = append(s.Link, fmt.Sprintf("/f/%s", size.Photo.Remote.Id))
		}
		cnt = append(cnt, s)

		return cnt
	case client.ConstructorMessageVideo:
		msg := content.(*client.MessageVideo)
		s := MessageAttachment{
			T:    msg.Video.GetConstructor(),
			Id:   msg.Video.Video.Remote.Id,
			Link: append(make([]string, 0), fmt.Sprintf("/f/%s", msg.Video.Video.Remote.Id)),
		}
		if msg.Video.Minithumbnail != nil {
			s.Thumb = base64.StdEncoding.EncodeToString(msg.Video.Minithumbnail.Data)
		}
		cnt = append(cnt, s)

		return cnt
	case client.ConstructorMessageAnimation:
		msg := content.(*client.MessageAnimation)
		s := MessageAttachment{
			T:    msg.Animation.GetConstructor(),
			Id:   msg.Animation.Animation.Remote.Id,
			Link: append(make([]string, 0), fmt.Sprintf("/f/%s", msg.Animation.Animation.Remote.Id)),
		}
		if msg.Animation.Minithumbnail != nil {
			s.Thumb = base64.StdEncoding.EncodeToString(msg.Animation.Minithumbnail.Data)
		}

		cnt = append(cnt, s)

		return cnt
	case client.ConstructorMessageSticker:
		msg := content.(*client.MessageSticker)
		if msg.Sticker.FullType != nil {
			s := MessageAttachment{
				T:    msg.Sticker.FullType.StickerFullTypeConstructor(),
				Id:   msg.Sticker.Sticker.Remote.Id,
				Link: append(make([]string, 0), fmt.Sprintf("/f/%s", msg.Sticker.Sticker.Remote.Id)),
				Name: msg.Sticker.FullType.StickerFullTypeConstructor(),
			}
			if msg.Sticker.Thumbnail != nil {
				s.ThumbLink = fmt.Sprintf("/f/%s", msg.Sticker.Thumbnail.File.Remote.Id)
			}
			cnt = append(cnt, s)

			return cnt
		}
		log.Printf("Invalid sticker in messsage (probably it's webp photo): %s", helpers.JsonMarshalStr(msg))

		return nil
	case client.ConstructorMessageVoiceNote:
		msg := content.(*client.MessageVoiceNote)
		s := MessageAttachment{
			T:    msg.VoiceNote.GetConstructor(),
			Id:   msg.VoiceNote.Voice.Remote.Id,
			Name: fmt.Sprintf("Voice (%ds.)", msg.VoiceNote.Duration),
			Link: append(make([]string, 0), fmt.Sprintf("/v/%s", msg.VoiceNote.Voice.Remote.Id)),
		}
		cnt = append(cnt, s)

		return cnt
	case client.ConstructorMessageVideoNote:
		msg := content.(*client.MessageVideoNote)
		s := MessageAttachment{
			T:    msg.VideoNote.GetConstructor(),
			Id:   msg.VideoNote.Video.Remote.Id,
			Name: fmt.Sprintf("Video note (%ds.)", msg.VideoNote.Duration),
			Link: append(make([]string, 0), fmt.Sprintf("/v/%s", msg.VideoNote.Video.Remote.Id)),
		}
		if msg.VideoNote.Minithumbnail != nil {
			s.Thumb = base64.StdEncoding.EncodeToString(msg.VideoNote.Minithumbnail.Data)
		}
		cnt = append(cnt, s)

		return cnt
	case client.ConstructorMessageDocument:
		msg := content.(*client.MessageDocument)
		s := MessageAttachment{
			T:    msg.Document.GetConstructor(),
			Id:   msg.Document.Document.Remote.Id,
			Name: msg.Document.FileName,
			Link: append(make([]string, 0), fmt.Sprintf("/f/%s", msg.Document.Document.Remote.Id)),
		}
		cnt = append(cnt, s)

		return cnt
	case client.ConstructorMessageAnimatedEmoji:
	//	msg := content.(*client.MessageAnimatedEmoji)
	//	s := MessageAttachment{
	//		T:    msg.AnimatedEmoji.Type,
	//		Id:   msg.AnimatedEmoji.Sticker.Sticker.Remote.Id,
	//		Name: msg.AnimatedEmoji.Sticker.Emoji,
	//		Link: append(make([]string, 0), fmt.Sprintf("/f/%s", msg.AnimatedEmoji.Sticker.Thumbnail.File.Remote)),
	//	}
	//	cnt = append(cnt, s)
	//
	//	return cnt

	case client.ConstructorMessageText:
	case client.ConstructorMessageChatChangeTitle:
	case client.ConstructorMessageChatChangePhoto:
	case client.ConstructorMessageCall:
	case client.ConstructorMessagePoll:
	case client.ConstructorMessageLocation:
	case client.ConstructorMessageChatAddMembers:
	case client.ConstructorMessageChatJoinByLink:
	case client.ConstructorMessageChatJoinByRequest:
	case client.ConstructorMessageChatDeleteMember:
	case client.ConstructorMessageBasicGroupChatCreate:
	case client.ConstructorMessagePinMessage:
	case client.ConstructorMessageAudio:
	case client.ConstructorMessageContact:
	case client.ConstructorMessageInvoice:
	case client.ConstructorMessageVideoChatEnded:
	case client.ConstructorMessageVideoChatStarted:
	case client.ConstructorMessageScreenshotTaken:
	case client.ConstructorMessageForumTopicEdited:

	case client.ConstructorMessageChatSetMessageAutoDeleteTime:
	case client.ConstructorMessageChatSetTheme:

	case client.ConstructorMessageUnsupported:

	default:
		log.Printf("Unknown content type: %s", cType)

		return nil
	}

	return nil
}
