package utils

import (
	"encoding/base64"
	"fmt"

	"github.com/alexbilevskiy/tgwatch/internal/helpers"
	"github.com/alexbilevskiy/tgwatch/internal/web/models"
	"github.com/zelenin/go-tdlib/client"
)

func GetContentWithText(content client.MessageContent, chatId int64) models.MessageTextContent {
	if content == nil {

		return models.MessageTextContent{Text: "UNSUPPORTED_CONTENT"}
	}

	cType := content.MessageContentConstructor()
	switch cType {
	case client.ConstructorMessageText:
		msg := content.(*client.MessageText)

		return models.MessageTextContent{FormattedText: msg.Text}
	case client.ConstructorMessagePhoto:
		msg := content.(*client.MessagePhoto)

		return models.MessageTextContent{FormattedText: msg.Caption}
	case client.ConstructorMessageVideo:
		msg := content.(*client.MessageVideo)

		return models.MessageTextContent{FormattedText: msg.Caption}
	case client.ConstructorMessageAnimation:
		msg := content.(*client.MessageAnimation)

		return models.MessageTextContent{FormattedText: msg.Caption}
	case client.ConstructorMessagePoll:
		msg := content.(*client.MessagePoll)

		return models.MessageTextContent{Text: fmt.Sprintf("Poll, %s", msg.Poll.Question)}
	case client.ConstructorMessageSticker:
		msg := content.(*client.MessageSticker)

		return models.MessageTextContent{Text: fmt.Sprintf("%s sticker", msg.Sticker.Emoji)}
	case client.ConstructorMessageVoiceNote:
		msg := content.(*client.MessageVoiceNote)

		return models.MessageTextContent{FormattedText: msg.Caption}
	case client.ConstructorMessageVideoNote:
		return models.MessageTextContent{Text: ""}
	case client.ConstructorMessageDocument:
		msg := content.(*client.MessageDocument)

		return models.MessageTextContent{FormattedText: msg.Caption}
	case client.ConstructorMessageChatAddMembers:
		msg := content.(*client.MessageChatAddMembers)

		return models.MessageTextContent{Text: fmt.Sprintf("Added users %s", helpers.JsonMarshalStr(msg.MemberUserIds))}
	case client.ConstructorMessagePinMessage:
		msg := content.(*client.MessagePinMessage)
		var url client.TextEntityType
		//@TODO: where to get chat ID?
		url = &client.TextEntityTypeTextUrl{Url: fmt.Sprintf("/m/%d/%d", chatId, msg.MessageId)}
		entity := &client.TextEntity{Type: url, Offset: 0, Length: 6}
		t := &client.FormattedText{Text: "Pinned message", Entities: append(make([]*client.TextEntity, 0), entity)}

		return models.MessageTextContent{FormattedText: t}
	case client.ConstructorMessageCall:
		msg := content.(*client.MessageCall)

		return models.MessageTextContent{Text: fmt.Sprintf("Call (%ds)", msg.Duration)}
	case client.ConstructorMessageAnimatedEmoji:
		msg := content.(*client.MessageAnimatedEmoji)
		if msg.AnimatedEmoji.Sticker != nil {

			return models.MessageTextContent{Text: fmt.Sprintf("%s (animated)", msg.AnimatedEmoji.Sticker.Emoji)}
		}
		return models.MessageTextContent{Text: "(invalid animated sticker)"}

	case client.ConstructorMessageChatChangeTitle:
		msg := content.(*client.MessageChatChangeTitle)

		return models.MessageTextContent{Text: fmt.Sprintf("Chat name was changed to '%s'", msg.Title)}
	case client.ConstructorMessageScreenshotTaken:

		return models.MessageTextContent{Text: "has taken screenshot!"}
	case client.ConstructorMessageChatJoinByLink:

		return models.MessageTextContent{Text: "joined by invite link"}
	case client.ConstructorMessageChatUpgradeTo:

		return models.MessageTextContent{Text: "chat upgraded to supergroup"}
	case client.ConstructorMessageForumTopicCreated:
		msg := content.(*client.MessageForumTopicCreated)

		return models.MessageTextContent{Text: fmt.Sprintf("topic created: `%s`", msg.Name)}
	case client.ConstructorMessageChatDeleteMember:
		msg := content.(*client.MessageChatDeleteMember)
		//@TODO: pass currentAcc as argument
		return models.MessageTextContent{Text: fmt.Sprintf("deleted `%d` from chat", msg.UserId)}
	case client.ConstructorMessageUnsupported:
		//msg := content.(*client.MessageUnsupported)
		return models.MessageTextContent{Text: ">unsupported message<"}
	default:
		fmt.Printf("unknown text type: %s\n", content.MessageContentConstructor())

		return models.MessageTextContent{Text: helpers.JsonMarshalStr(content)}
	}
}

func GetContentAttachments(content client.MessageContent) []models.MessageAttachment {
	if content == nil {

		return nil
	}
	cType := content.MessageContentConstructor()
	var cnt []models.MessageAttachment
	switch cType {
	case client.ConstructorMessagePhoto:
		msg := content.(*client.MessagePhoto)
		s := models.MessageAttachment{
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
		s := models.MessageAttachment{
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
		s := models.MessageAttachment{
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
			s := models.MessageAttachment{
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
		fmt.Printf("Invalid sticker in messsage (probably it's webp photo): %s\n", helpers.JsonMarshalStr(msg))

		return nil
	case client.ConstructorMessageVoiceNote:
		msg := content.(*client.MessageVoiceNote)
		s := models.MessageAttachment{
			T:    msg.VoiceNote.GetConstructor(),
			Id:   msg.VoiceNote.Voice.Remote.Id,
			Name: fmt.Sprintf("Voice (%ds.)", msg.VoiceNote.Duration),
			Link: append(make([]string, 0), fmt.Sprintf("/v/%s", msg.VoiceNote.Voice.Remote.Id)),
		}
		cnt = append(cnt, s)

		return cnt
	case client.ConstructorMessageVideoNote:
		msg := content.(*client.MessageVideoNote)
		s := models.MessageAttachment{
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
		s := models.MessageAttachment{
			T:    msg.Document.GetConstructor(),
			Id:   msg.Document.Document.Remote.Id,
			Name: msg.Document.FileName,
			Link: append(make([]string, 0), fmt.Sprintf("/f/%s", msg.Document.Document.Remote.Id)),
		}
		cnt = append(cnt, s)

		return cnt
	case client.ConstructorMessageAnimatedEmoji:
	//	msg := content.(*client.MessageAnimatedEmoji)
	//	s := models.MessageAttachment{
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
	case client.ConstructorMessageChatUpgradeTo:
	case client.ConstructorMessageChatUpgradeFrom:
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
	case client.ConstructorMessageForumTopicCreated:

	case client.ConstructorMessageChatSetMessageAutoDeleteTime:
	case client.ConstructorMessageChatSetTheme:

	case client.ConstructorMessageUnsupported:

	default:
		fmt.Printf("Unknown content type: %s\n", cType)

		return nil
	}

	return nil
}
