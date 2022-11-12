package libs

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/alexbilevskiy/tgWatch/pkg/config"
	"github.com/alexbilevskiy/tgWatch/pkg/structs"
	"github.com/zelenin/go-tdlib/client"
	"go.mongodb.org/mongo-driver/bson"
	"log"
	"strconv"
	"sync"
)

func GetChatIdBySender(sender client.MessageSender) int64 {
	senderChatId := int64(0)
	if sender.MessageSenderType() == "messageSenderChat" {
		senderChatId = sender.(*client.MessageSenderChat).ChatId
	} else if sender.MessageSenderType() == "messageSenderUser" {
		senderChatId = int64(sender.(*client.MessageSenderUser).UserId)
	}

	return senderChatId
}

func GetSenderName(acc int64, sender client.MessageSender) string {
	chat, err := GetSenderObj(acc, sender)
	if err != nil {

		return err.Error()
	}
	if sender.MessageSenderType() == "messageSenderChat" {
		name := fmt.Sprintf("%s", chat.(*client.Chat).Title)
		if name == "" {
			name = fmt.Sprintf("no_name %d", chat.(*client.Chat).Id)
		}
		return name
	} else if sender.MessageSenderType() == "messageSenderUser" {
		user := chat.(*client.User)
		return getUserFullname(user)
	}

	return "unkown_chattype"
}

func getUserFullname(user *client.User) string {
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
		log.Printf("whoa, multiple usernames? %s", JsonMarshalStr(usernames.ActiveUsernames))
		return usernames.ActiveUsernames[0]
	}

	return usernames.ActiveUsernames[0]
}

func GetSenderObj(acc int64, sender client.MessageSender) (interface{}, error) {
	if sender.MessageSenderType() == "messageSenderChat" {
		chatId := sender.(*client.MessageSenderChat).ChatId
		chat, err := GetChat(acc, chatId, false)
		if err != nil {
			log.Printf("Failed to request sender chat info by id %d: %s", chatId, err)

			return nil, errors.New("unknown chat")
		}

		return chat, nil
	} else if sender.MessageSenderType() == "messageSenderUser" {
		userId := sender.(*client.MessageSenderUser).UserId
		user, err := GetUser(acc, userId)
		if err != nil {
			log.Printf("Failed to request user info by id %d: %s", userId, err)

			return nil, errors.New("unknown user")
		}

		return user, nil
	}

	return nil, errors.New("unknown sender type")
}

func GetChatName(acc int64, chatId int64) string {
	fullChat, err := GetChat(acc, chatId, false)
	if err != nil {
		log.Printf("Failed to get chat name by id %d: %s", chatId, err)

		return "no_title"
	}
	name := fmt.Sprintf("%s", fullChat.Title)
	if name == "" {
		name = fmt.Sprintf("no_name %d", chatId)
	}

	return name
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

		return structs.MessageTextContent{Text: fmt.Sprintf("Added users %s", JsonMarshalStr(msg.MemberUserIds))}
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

		return structs.MessageTextContent{Text: fmt.Sprintf("%s (animated)", msg.AnimatedEmoji.Sticker.Emoji)}
	case client.TypeMessageChatChangeTitle:
		msg := content.(*client.MessageChatChangeTitle)

		return structs.MessageTextContent{Text: fmt.Sprintf("Chat name was changed to '%s'", msg.Title)}
	case client.TypeMessageScreenshotTaken:

		return structs.MessageTextContent{Text: "has taken screenshot!"}
	case client.TypeMessageChatJoinByLink:

		return structs.MessageTextContent{Text: "joined by invite link"}
	case client.TypeMessageChatDeleteMember:
		msg := content.(*client.MessageChatDeleteMember)
		return structs.MessageTextContent{Text: fmt.Sprintf("deleted `%s` from chat", GetChatName(currentAcc, msg.UserId))}
	case client.TypeMessageUnsupported:
		//msg := content.(*client.MessageUnsupported)
		return structs.MessageTextContent{Text: ">unsupported message<"}
	default:
		log.Printf("unknown text type: %s", content.MessageContentType())

		return structs.MessageTextContent{Text: JsonMarshalStr(content)}
	}
}

func MarkJoinAsRead(acc int64, chatId int64, messageId int64) {
	chat, err := GetChat(acc, chatId, true)
	if err != nil {
		fmt.Printf("Cannot update unread count because chat %d not found: %s\n", chatId, err.Error())

		return
	}
	name := GetChatName(acc, chatId)

	if chat.UnreadCount != 1 {
		DLog(fmt.Sprintf("Chat `%s` %d unread count: %d>1, not marking as read\n", name, chatId, chat.UnreadCount))
		return
	}
	DLog(fmt.Sprintf("Chat `%s` %d unread count: %d, marking join as read\n", name, chatId, chat.UnreadCount))

	err = markAsRead(acc, chatId, messageId)
	if err != nil {
		fmt.Printf("Cannot mark as read chat %d, message %d: %s\n", chatId, messageId, err.Error())

		return
	}
	chat, err = GetChat(acc, chatId, true)
	if err != nil {
		fmt.Printf("Cannot get NEW unread count because chat %d not found: %s\n", chatId, err.Error())

		return
	}
	DLog(fmt.Sprintf("NEW Chat `%s` %d unread count: %d\n", name, chatId, chat.UnreadCount))

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
		if msg.Sticker.Type != nil {
			s := structs.MessageAttachment{
				T:    msg.Sticker.Type.StickerTypeType(),
				Id:   msg.Sticker.Sticker.Remote.Id,
				Link: append(make([]string, 0), fmt.Sprintf("http://%s/f/%s", config.Config.WebListen, msg.Sticker.Sticker.Remote.Id)),
				Name: msg.Sticker.Type.StickerTypeType(),
			}
			if msg.Sticker.Thumbnail != nil {
				s.ThumbLink = fmt.Sprintf("http://%s/f/%s", config.Config.WebListen, msg.Sticker.Thumbnail.File.Remote.Id)
			}
			cnt = append(cnt, s)

			return cnt
		}
		log.Printf("Invalid sticker in messsage (probably it's webp photo): %s", JsonMarshalStr(msg))

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

	case client.TypeMessageChatSetTtl:
	case client.TypeMessageChatSetTheme:

	case client.TypeMessageUnsupported:

	default:
		log.Printf("Unknown content type: %s", cType)

		return nil
	}

	return nil
}

func loadChatsList(acc int64, listId int32) {
	var chatList client.ChatList
	switch listId {
	case ClMain:
		chatList = &client.ChatListMain{}
	case ClArchive:
		chatList = &client.ChatListArchive{}
	default:
		chatList = &client.ChatListFilter{ChatFilterId: listId}
	}
	crit := bson.D{{"listid", listId}}
	d, err := chatListColl[acc].DeleteMany(mongoContext, crit)
	if err != nil {
		log.Printf("Failed to delete chats by list %d: %s\n", listId, err.Error())
	} else {
		log.Printf("Deleted %d chats by listid %d because refresh was called\n", d.DeletedCount, listId)
	}

	log.Printf("Requesting LoadChats for list %s id:%d", chatList.ChatListType(), listId)
	err = loadChats(acc, chatList)
	if err != nil {
		//@see https://github.com/tdlib/td/blob/fb39e5d74667db915a75a5e58065c59af8e7d8d6/td/generate/scheme/td_api.tl#L4171
		if err.Error() == "404 Not Found" {
			log.Printf("All chats already loaded")
		} else {
			log.Fatalf("[ERROR] LoadChats: %s", err)
		}
	}
}

func checkSkippedChat(acc int64, chatId string) bool {
	if _, ok := ignoreLists[acc].IgnoreAuthorIds[chatId]; ok {

		return true
	}
	if _, ok := ignoreLists[acc].IgnoreChatIds[chatId]; ok {

		return true
	}

	return false
}

func checkSkippedSenderBySavedMessage(acc int64, chatId int64, messageId int64) bool {
	savedMessage, err := FindUpdateNewMessage(acc, chatId, messageId)
	if err != nil {

		return false
	}

	if checkSkippedChat(acc, strconv.FormatInt(GetChatIdBySender(savedMessage.Message.SenderId), 10)) {

		return true
	}

	return false
}

func checkChatFilter(acc int64, chatId int64) bool {
	for _, filter := range chatFilters[acc] {
		for _, chatInFilter := range filter.IncludedChats {
			if chatInFilter == chatId && ignoreLists[acc].IgnoreFolders[filter.Title] {
				//log.Printf("Skip chat %d because it's in skipped folder %s", chatId, filter.Title)

				return true
			}
		}
	}

	return false
}

func SaveChatFilters(acc int64, chatFiltersUpdate *client.UpdateChatFilters) {
	log.Printf("Chat filters update! %s", chatFiltersUpdate.Type)
	//ClearChatFilters(acc)
	var wg sync.WaitGroup

	for _, filterInfo := range chatFiltersUpdate.ChatFilters {
		existed := false
		for _, existningFilter := range chatFilters[acc] {
			if existningFilter.Id == filterInfo.Id {
				existed = true
				break
			}
		}
		if existed {
			log.Printf("Existing chat filter: id: %d, n: %s", filterInfo.Id, filterInfo.Title)
			continue
		}
		log.Printf("New chat filter: id: %d, n: %s", filterInfo.Id, filterInfo.Title)

		wg.Add(1)
		go func(filterInfo *client.ChatFilterInfo, wg *sync.WaitGroup) {
			defer wg.Done()
			chatFilter, err := getChatFilter(acc, filterInfo.Id)
			if err != nil {
				log.Printf("Failed to load chat filter: id: %d, n: %s, reason: %s", filterInfo.Id, filterInfo.Title, err.Error())

				return
			}
			saveChatFilter(acc, chatFilter, filterInfo)
			log.Printf("Chat filter LOADED: id: %d, n: %s", filterInfo.Id, filterInfo.Title)
		}(filterInfo, &wg)
		//time.Sleep(time.Second * 2)
	}
	wg.Wait()

	for _, existningFilter := range chatFilters[acc] {
		deleted := true
		for _, filterInfo := range chatFiltersUpdate.ChatFilters {
			if filterInfo.Id == existningFilter.Id {
				deleted = false
				continue
			}
		}
		if !deleted {
			continue
		}
		log.Printf("Deleted chat filter: id: %d, n: %s", existningFilter.Id, existningFilter.Title)
		//@TODO: delete it
	}

	LoadChatFilters(acc)

}

func loadOptionsList(acc int64) {
	var opts map[string]structs.TdlibOption
	opts = make(map[string]structs.TdlibOption)
	config.UnmarshalJsonFile("tdlib_options.json", &opts)
	tdlibOptions[acc] = opts
}
