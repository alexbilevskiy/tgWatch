package tdlib

import (
	"encoding/base64"
	"fmt"
	"github.com/alexbilevskiy/tgWatch/pkg/config"
	"github.com/alexbilevskiy/tgWatch/pkg/libs/helpers"
	"github.com/alexbilevskiy/tgWatch/pkg/structs"
	"github.com/zelenin/go-tdlib/client"
	"log"
)

var TdlibOptions map[string]structs.TdlibOption

func GetChatIdBySender(sender client.MessageSender) int64 {
	senderChatId := int64(0)
	if sender.MessageSenderType() == "messageSenderChat" {
		senderChatId = sender.(*client.MessageSenderChat).ChatId
	} else if sender.MessageSenderType() == "messageSenderUser" {
		senderChatId = int64(sender.(*client.MessageSenderUser).UserId)
	}

	return senderChatId
}

func GetUserFullname(user *client.User) string {
	name := ""
	if user.FirstName != "" {
		name = user.FirstName
	}
	if user.LastName != "" {
		name = fmt.Sprintf("%s %s", name, user.LastName)
	}
	un := GetUsername(user.Usernames)
	if un != "" {
		name = fmt.Sprintf("%s (@%s)", name, un)
	}
	if name == "" {
		name = fmt.Sprintf("no_name %d", user.Id)
	}
	return name
}

func GetUsername(usernames *client.Usernames) string {
	if usernames == nil {
		return ""
	}
	if len(usernames.ActiveUsernames) == 0 {
		return ""
	}
	if len(usernames.ActiveUsernames) > 1 {
		log.Printf("whoa, multiple usernames? %s", helpers.JsonMarshalStr(usernames.ActiveUsernames))
		return usernames.ActiveUsernames[0]
	}

	return usernames.ActiveUsernames[0]
}

func GetContentWithText(content client.MessageContent, chatId int64) structs.MessageTextContent {
	if content == nil {

		return structs.MessageTextContent{Text: "UNSUPPORTED_CONTENT"}
	}

	cType := content.MessageContentType()
	switch cType {
	case client.TypeMessageText:
		msg := content.(*client.MessageText)

		return structs.MessageTextContent{FormattedText: msg.Text}
	case client.TypeMessagePhoto:
		msg := content.(*client.MessagePhoto)

		return structs.MessageTextContent{FormattedText: msg.Caption}
	case client.TypeMessageVideo:
		msg := content.(*client.MessageVideo)

		return structs.MessageTextContent{FormattedText: msg.Caption}
	case client.TypeMessageAnimation:
		msg := content.(*client.MessageAnimation)

		return structs.MessageTextContent{FormattedText: msg.Caption}
	case client.TypeMessagePoll:
		msg := content.(*client.MessagePoll)

		return structs.MessageTextContent{Text: fmt.Sprintf("Poll, %s", msg.Poll.Question)}
	case client.TypeMessageSticker:
		msg := content.(*client.MessageSticker)

		return structs.MessageTextContent{Text: fmt.Sprintf("%s sticker", msg.Sticker.Emoji)}
	case client.TypeMessageVoiceNote:
		msg := content.(*client.MessageVoiceNote)

		return structs.MessageTextContent{FormattedText: msg.Caption}
	case client.TypeMessageVideoNote:
		return structs.MessageTextContent{Text: ""}
	case client.TypeMessageDocument:
		msg := content.(*client.MessageDocument)

		return structs.MessageTextContent{FormattedText: msg.Caption}
	case client.TypeMessageChatAddMembers:
		msg := content.(*client.MessageChatAddMembers)

		return structs.MessageTextContent{Text: fmt.Sprintf("Added users %s", helpers.JsonMarshalStr(msg.MemberUserIds))}
	case client.TypeMessagePinMessage:
		msg := content.(*client.MessagePinMessage)
		var url client.TextEntityType
		//@TODO: where to get chat ID?
		url = &client.TextEntityTypeTextUrl{Url: fmt.Sprintf("/m/%d/%d", chatId, msg.MessageId)}
		entity := &client.TextEntity{Type: url, Offset: 0, Length: 6}
		t := &client.FormattedText{Text: "Pinned message", Entities: append(make([]*client.TextEntity, 0), entity)}

		return structs.MessageTextContent{FormattedText: t}
	case client.TypeMessageCall:
		msg := content.(*client.MessageCall)

		return structs.MessageTextContent{Text: fmt.Sprintf("Call (%ds)", msg.Duration)}
	case client.TypeMessageAnimatedEmoji:
		msg := content.(*client.MessageAnimatedEmoji)
		if msg.AnimatedEmoji.Sticker != nil {

			return structs.MessageTextContent{Text: fmt.Sprintf("%s (animated)", msg.AnimatedEmoji.Sticker.Emoji)}
		}
		return structs.MessageTextContent{Text: "(invalid animated sticker)"}

	case client.TypeMessageChatChangeTitle:
		msg := content.(*client.MessageChatChangeTitle)

		return structs.MessageTextContent{Text: fmt.Sprintf("Chat name was changed to '%s'", msg.Title)}
	case client.TypeMessageScreenshotTaken:

		return structs.MessageTextContent{Text: "has taken screenshot!"}
	case client.TypeMessageChatJoinByLink:

		return structs.MessageTextContent{Text: "joined by invite link"}
	case client.TypeMessageChatDeleteMember:
		msg := content.(*client.MessageChatDeleteMember)
		//@TODO: pass currentAcc as argument
		return structs.MessageTextContent{Text: fmt.Sprintf("deleted `%d` from chat", msg.UserId)}
	case client.TypeMessageUnsupported:
		//msg := content.(*client.MessageUnsupported)
		return structs.MessageTextContent{Text: ">unsupported message<"}
	default:
		log.Printf("unknown text type: %s", content.MessageContentType())

		return structs.MessageTextContent{Text: helpers.JsonMarshalStr(content)}
	}
}

func GetContentAttachments(content client.MessageContent) []structs.MessageAttachment {
	if content == nil {

		return nil
	}
	cType := content.MessageContentType()
	var cnt []structs.MessageAttachment
	switch cType {
	case client.TypeMessagePhoto:
		msg := content.(*client.MessagePhoto)
		s := structs.MessageAttachment{
			T:  msg.Photo.Type,
			Id: msg.Photo.Sizes[len(msg.Photo.Sizes)-1].Photo.Remote.Id,
		}
		if msg.Photo.Minithumbnail != nil {
			s.Thumb = base64.StdEncoding.EncodeToString(msg.Photo.Minithumbnail.Data)
		}
		for _, size := range msg.Photo.Sizes {
			s.Link = append(s.Link, fmt.Sprintf("http://%s/f/%s", config.Config.WebListen, size.Photo.Remote.Id))
		}
		cnt = append(cnt, s)

		return cnt
	case client.TypeMessageVideo:
		msg := content.(*client.MessageVideo)
		s := structs.MessageAttachment{
			T:    msg.Video.Type,
			Id:   msg.Video.Video.Remote.Id,
			Link: append(make([]string, 0), fmt.Sprintf("http://%s/f/%s", config.Config.WebListen, msg.Video.Video.Remote.Id)),
		}
		if msg.Video.Minithumbnail != nil {
			s.Thumb = base64.StdEncoding.EncodeToString(msg.Video.Minithumbnail.Data)
		}
		cnt = append(cnt, s)

		return cnt
	case client.TypeMessageAnimation:
		msg := content.(*client.MessageAnimation)
		s := structs.MessageAttachment{
			T:    msg.Animation.Type,
			Id:   msg.Animation.Animation.Remote.Id,
			Link: append(make([]string, 0), fmt.Sprintf("http://%s/f/%s", config.Config.WebListen, msg.Animation.Animation.Remote.Id)),
		}
		if msg.Animation.Minithumbnail != nil {
			s.Thumb = base64.StdEncoding.EncodeToString(msg.Animation.Minithumbnail.Data)
		}

		cnt = append(cnt, s)

		return cnt
	case client.TypeMessageSticker:
		msg := content.(*client.MessageSticker)
		if msg.Sticker.FullType != nil {
			s := structs.MessageAttachment{
				T:    msg.Sticker.FullType.StickerFullTypeType(),
				Id:   msg.Sticker.Sticker.Remote.Id,
				Link: append(make([]string, 0), fmt.Sprintf("http://%s/f/%s", config.Config.WebListen, msg.Sticker.Sticker.Remote.Id)),
				Name: msg.Sticker.FullType.StickerFullTypeType(),
			}
			if msg.Sticker.Thumbnail != nil {
				s.ThumbLink = fmt.Sprintf("http://%s/f/%s", config.Config.WebListen, msg.Sticker.Thumbnail.File.Remote.Id)
			}
			cnt = append(cnt, s)

			return cnt
		}
		log.Printf("Invalid sticker in messsage (probably it's webp photo): %s", helpers.JsonMarshalStr(msg))

		return nil
	case client.TypeMessageVoiceNote:
		msg := content.(*client.MessageVoiceNote)
		s := structs.MessageAttachment{
			T:    msg.VoiceNote.Type,
			Id:   msg.VoiceNote.Voice.Remote.Id,
			Name: fmt.Sprintf("Voice (%ds.)", msg.VoiceNote.Duration),
			Link: append(make([]string, 0), fmt.Sprintf("http://%s/v/%s", config.Config.WebListen, msg.VoiceNote.Voice.Remote.Id)),
		}
		cnt = append(cnt, s)

		return cnt
	case client.TypeMessageVideoNote:
		msg := content.(*client.MessageVideoNote)
		s := structs.MessageAttachment{
			T:    msg.VideoNote.Type,
			Id:   msg.VideoNote.Video.Remote.Id,
			Name: fmt.Sprintf("Video note (%ds.)", msg.VideoNote.Duration),
			Link: append(make([]string, 0), fmt.Sprintf("http://%s/v/%s", config.Config.WebListen, msg.VideoNote.Video.Remote.Id)),
		}
		if msg.VideoNote.Minithumbnail != nil {
			s.Thumb = base64.StdEncoding.EncodeToString(msg.VideoNote.Minithumbnail.Data)
		}
		cnt = append(cnt, s)

		return cnt
	case client.TypeMessageDocument:
		msg := content.(*client.MessageDocument)
		s := structs.MessageAttachment{
			T:    msg.Document.Type,
			Id:   msg.Document.Document.Remote.Id,
			Name: msg.Document.FileName,
			Link: append(make([]string, 0), fmt.Sprintf("http://%s/f/%s", config.Config.WebListen, msg.Document.Document.Remote.Id)),
		}
		cnt = append(cnt, s)

		return cnt
	case client.TypeMessageAnimatedEmoji:
	//	msg := content.(*client.MessageAnimatedEmoji)
	//	s := structs.MessageAttachment{
	//		T:    msg.AnimatedEmoji.Type,
	//		Id:   msg.AnimatedEmoji.Sticker.Sticker.Remote.Id,
	//		Name: msg.AnimatedEmoji.Sticker.Emoji,
	//		Link: append(make([]string, 0), fmt.Sprintf("http://%s/f/%s", config.Config.WebListen, msg.AnimatedEmoji.Sticker.Thumbnail.File.Remote)),
	//	}
	//	cnt = append(cnt, s)
	//
	//	return cnt

	case client.TypeMessageText:
	case client.TypeMessageChatChangeTitle:
	case client.TypeMessageChatChangePhoto:
	case client.TypeMessageCall:
	case client.TypeMessagePoll:
	case client.TypeMessageLocation:
	case client.TypeMessageChatAddMembers:
	case client.TypeMessageChatJoinByLink:
	case client.TypeMessageChatJoinByRequest:
	case client.TypeMessageChatDeleteMember:
	case client.TypeMessageBasicGroupChatCreate:
	case client.TypeMessagePinMessage:
	case client.TypeMessageAudio:
	case client.TypeMessageContact:
	case client.TypeMessageInvoice:
	case client.TypeMessageVideoChatEnded:
	case client.TypeMessageVideoChatStarted:
	case client.TypeMessageScreenshotTaken:
	case client.TypeMessageForumTopicEdited:

	case client.TypeMessageChatSetMessageAutoDeleteTime:
	case client.TypeMessageChatSetTheme:

	case client.TypeMessageUnsupported:

	default:
		log.Printf("Unknown content type: %s", cType)

		return nil
	}

	return nil
}

func LoadOptionsList() {
	var opts map[string]structs.TdlibOption
	opts = make(map[string]structs.TdlibOption)
	config.UnmarshalJsonFile("tdlib_options.json", &opts)
	TdlibOptions = opts
}
