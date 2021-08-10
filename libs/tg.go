package libs

import (
	"encoding/base64"
	"errors"
	"fmt"
	"go-tdlib/client"
	"go.mongodb.org/mongo-driver/bson"
	"log"
	"path/filepath"
	"strconv"
	"tgWatch/config"
	"tgWatch/structs"
	"time"
)

func initTdlib() {
	LoadSettings()
	LoadChatFilters()
	loadOptionsList()
	localChats = make(map[int64]*client.Chat)
	authorizer := client.ClientAuthorizer()
	go client.CliInteractor(authorizer)

	authorizer.TdlibParameters <- &client.TdlibParameters{
		UseTestDc:              false,
		DatabaseDirectory:      filepath.Join(".tdlib", "database"),
		FilesDirectory:         filepath.Join(".tdlib", "files"),
		UseFileDatabase:        true,
		UseChatInfoDatabase:    true,
		UseMessageDatabase:     true,
		UseSecretChats:         false,
		ApiId:                  config.Config.ApiId,
		ApiHash:                config.Config.ApiHash,
		SystemLanguageCode:     "en",
		DeviceModel:            "Linux",
		SystemVersion:          "1.0.0",
		ApplicationVersion:     "1.0.0",
		EnableStorageOptimizer: true,
		IgnoreFileNames:        false,
	}

	logVerbosity := client.WithLogVerbosity(&client.SetLogVerbosityLevelRequest{
		NewVerbosityLevel: 0,
	})

	var err error
	tdlibClient, err = client.NewClient(authorizer, logVerbosity)
	if err != nil {
		log.Fatalf("NewClient error: %s", err)
	}

	optionValue, err := tdlibClient.GetOption(&client.GetOptionRequest{
		Name: "version",
	})
	if err != nil {
		log.Fatalf("GetOption error: %s", err)
	}

	log.Printf("TDLib version: %s", optionValue.(*client.OptionValueString).Value)

	me, err = tdlibClient.GetMe()
	if err != nil {
		log.Fatalf("GetMe error: %s", err)
	}

	log.Printf("Me: %s %s [%s]", me.FirstName, me.LastName, me.Username)

	//@NOTE: https://github.com/tdlib/td/issues/1005#issuecomment-613839507
	go func() {
		//for true {
		{
			req := &client.SetOptionRequest{Name: "online", Value: &client.OptionValueBoolean{Value: true}}
			ok, err := tdlibClient.SetOption(req)
			if err != nil {
				log.Printf("failed to set online option: %s", err)
			} else {
				log.Printf("Set online status: %s", JsonMarshalStr(ok))
			}
			//time.Sleep(10 * time.Second)
		}
	}()

	//req := &client.SetOptionRequest{Name: "ignore_background_updates", Value: &client.OptionValueBoolean{Value: false}}
	//ok, err := tdlibClient.SetOption(req)
	//if err != nil {
	//	log.Printf("failed to set ignore_background_updates option: %s", err)
	//} else {
	//	log.Printf("Set ignore_background_updates option: %s", JsonMarshalStr(ok))
	//}

}

func GetChatIdBySender(sender client.MessageSender) int64 {
	senderChatId := int64(0)
	if sender.MessageSenderType() == "messageSenderChat" {
		senderChatId = sender.(*client.MessageSenderChat).ChatId
	} else if sender.MessageSenderType() == "messageSenderUser" {
		senderChatId = int64(sender.(*client.MessageSenderUser).UserId)
	}

	return senderChatId
}

func GetSenderName(sender client.MessageSender) string {
	chat, err := GetSenderObj(sender)
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
	if user.Username != "" {
		name = fmt.Sprintf("%s (@%s)", name, user.Username)
	}
	if name == "" {
		name = fmt.Sprintf("no_name %d", user.Id)
	}
	return name
}

func GetSenderObj(sender client.MessageSender) (interface{}, error) {
	if sender.MessageSenderType() == "messageSenderChat" {
		chatId := sender.(*client.MessageSenderChat).ChatId
		chat, err := GetChat(chatId, false)
		if err != nil {
			log.Printf("Failed to request sender chat info by id %d: %s", chatId, err)

			return nil, errors.New("unknown chat")
		}

		return chat, nil
	} else if sender.MessageSenderType() == "messageSenderUser" {
		userId := sender.(*client.MessageSenderUser).UserId
		user, err := GetUser(userId)
		if err != nil {
			log.Printf("Failed to request user info by id %d: %s", userId, err)

			return nil, errors.New("unknown user")
		}

		return user, nil
	}

	return nil, errors.New("unknown sender type")
}

func GetLink(chatId int64, messageId int64) string {
	linkReq := &client.GetMessageLinkRequest{ChatId: chatId, MessageId: messageId}
	link, err := tdlibClient.GetMessageLink(linkReq)
	if err != nil {
		if err.Error() != "400 Public message links are available only for messages in supergroups and channel chats" {
			log.Printf("Failed to get msg link by chat id %d, msg id %d: %s", chatId, messageId, err)
		}

		return ""
	}

	return link.Link
}

func GetChatName(chatId int64) string {
	fullChat, err := GetChat(chatId, false)
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

func GetChat(chatId int64, force bool) (*client.Chat, error) {
	fullChat, ok := localChats[chatId]
	if !force && ok {
		//fmt.Printf("Found local chat %d\n", chatId)

		return fullChat, nil
	}
	req := &client.GetChatRequest{ChatId: chatId}
	fullChat, err := tdlibClient.GetChat(req)
	if err == nil {
		DLog(fmt.Sprintf("Caching local chat %d\n", chatId))
		localChats[chatId] = fullChat
	}

	return fullChat, err
}

func GetUser(userId int32) (*client.User, error) {
	userReq := &client.GetUserRequest{UserId: userId}

	return tdlibClient.GetUser(userReq)
}

func GetSuperGroup(sgId int32) (*client.Supergroup, error) {
	sgReq := &client.GetSupergroupRequest{SupergroupId: sgId}

	return tdlibClient.GetSupergroup(sgReq)
}

func GetBasicGroup(groupId int32) (*client.BasicGroup, error) {
	bgReq := &client.GetBasicGroupRequest{BasicGroupId: groupId}

	return tdlibClient.GetBasicGroup(bgReq)
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

		return structs.MessageTextContent{Text: fmt.Sprintf("Sticker, %s", msg.Sticker.Emoji)}
	case client.TypeMessageVoiceNote:
		msg := content.(*client.MessageVoiceNote)

		return structs.MessageTextContent{FormattedText: msg.Caption}
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
	default:

		return structs.MessageTextContent{Text: JsonMarshalStr(content)}
	}
}

func MarkAsReadMessage(chatId int64, messageId int64) {
	chat, err := GetChat(chatId, true)
	if err != nil {
		fmt.Printf("Cannot update unread count because chat %d not found: %s\n", chatId, err.Error())

		return
	}
	name := GetChatName(chatId)

	if chat.UnreadCount != 1 {
		DLog(fmt.Sprintf("Chat `%s` %d unread count: %d>1, not marking as read\n", name, chatId, chat.UnreadCount))
		return
	}
	fmt.Printf("Chat `%s` %d unread count: %d, marking join as read\n", name, chatId, chat.UnreadCount)

	req := &client.ViewMessagesRequest{ChatId: chatId, MessageIds: append(make([]int64, 0), messageId), ForceRead: true}
	_, err = tdlibClient.ViewMessages(req)
	if err != nil {
		fmt.Printf("Cannot mark as read chat %d, message %d: %s\n", chatId, messageId, err.Error())

		return
	}
	chat, err = GetChat(chatId, true)
	if err != nil {
		fmt.Printf("Cannot get NEW unread count because chat %d not found: %s\n", chatId, err.Error())

		return
	}
	DLog(fmt.Sprintf("NEW Chat `%s` %d unread count: %d\n", name, chatId, chat.UnreadCount))

}

func DownloadFile(id int32) (*client.File, error) {
	req := client.DownloadFileRequest{FileId: id, Priority: 1, Synchronous: true}
	file, err := tdlibClient.DownloadFile(&req)
	if err != nil {
		log.Printf("Cannot download file: %s %d", err, id)

		return nil, err
	}

	return file, nil
}

func DownloadFileByRemoteId(id string) (*client.File, error) {
	remoteFileReq := client.GetRemoteFileRequest{RemoteFileId: id}
	remoteFile, err := tdlibClient.GetRemoteFile(&remoteFileReq)
	if err != nil {
		log.Printf("Cannot download remote file: %s %s", err, id)

		return nil, err
	}

	return DownloadFile(remoteFile.Id)
}

func GetContentAttachments(content client.MessageContent) []structs.MessageAttachment {
	if content == nil {

		return nil
	}
	cType := content.MessageContentType()
	var cnt []structs.MessageAttachment
	switch cType {
	case client.TypeMessageText:
	case client.TypeMessageCall:

		return nil
	case client.TypeMessagePhoto:
		msg := content.(*client.MessagePhoto)
		s := structs.MessageAttachment{
			T: msg.Photo.Type,
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
			T: msg.Video.Type,
			Id: msg.Video.Video.Remote.Id,
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
			T: msg.Animation.Type,
			Id: msg.Animation.Animation.Remote.Id,
			Link: append(make([]string, 0), fmt.Sprintf("http://%s/f/%s", config.Config.WebListen, msg.Animation.Animation.Remote.Id)),
		}
		if msg.Animation.Minithumbnail != nil {
			s.Thumb = base64.StdEncoding.EncodeToString(msg.Animation.Minithumbnail.Data)
		}

		cnt = append(cnt, s)

		return cnt
	case client.TypeMessageSticker:
		msg := content.(*client.MessageSticker)
		s := structs.MessageAttachment{
			T: msg.Sticker.Type,
			Id: msg.Sticker.Sticker.Remote.Id,
			Link: append(make([]string, 0), fmt.Sprintf("http://%s/f/%s", config.Config.WebListen, msg.Sticker.Sticker.Remote.Id)),
		}
		cnt = append(cnt, s)

		return cnt
	case client.TypeMessageVoiceNote:
		msg := content.(*client.MessageVoiceNote)
		s := structs.MessageAttachment{
			T: msg.VoiceNote.Type,
			Id: msg.VoiceNote.Voice.Remote.Id,
			Name: fmt.Sprintf("Voice (%ds.)", msg.VoiceNote.Duration),
			Link: append(make([]string, 0), fmt.Sprintf("http://%s/f/%s", config.Config.WebListen, msg.VoiceNote.Voice.Remote.Id)),
		}
		cnt = append(cnt, s)

		return cnt
	case client.TypeMessageDocument:
		msg := content.(*client.MessageDocument)
		s := structs.MessageAttachment{
			T: msg.Document.Type,
			Id: msg.Document.Document.Remote.Id,
			Name: msg.Document.FileName,
			Link: append(make([]string, 0), fmt.Sprintf("http://%s/f/%s", config.Config.WebListen, msg.Document.Document.Remote.Id)),
		}
		cnt = append(cnt, s)

		return cnt
	case client.TypeMessagePoll:
	case client.TypeMessageLocation:
	case client.TypeMessageChatAddMembers:
	case client.TypeMessageChatJoinByLink:
	case client.TypeMessageBasicGroupChatCreate:
	case client.TypeMessagePinMessage:
	case client.TypeMessageVideoNote:
	case client.TypeMessageAudio:
	case client.TypeMessageContact:
	case client.TypeMessageInvoice:

	default:
		log.Printf("Unknown content type: %s", cType)

		return nil
	}

	return nil
}

func getChatsList(listId int32) []*client.Chat {
	maxChatId := client.JsonInt64(int64((^uint64(0)) >> 1))
	offsetOrder := maxChatId
	log.Printf("Requesting chats with max id: %d", maxChatId)

	var fullList []*client.Chat

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
	d, err := chatListColl.DeleteMany(mongoContext, crit)
	if err != nil {
		fmt.Printf("Failed to delete chats by list %d: %s\n", listId, err.Error())
	} else {
		fmt.Printf("Deleted %d chats by listid %d\n", d.DeletedCount, listId)
	}

	page := 0
	offsetChatId := int64(0)
	for {
		log.Printf("GetChats requesting page %d, offset %d", page, offsetChatId)
		chatsRequest := &client.GetChatsRequest{ChatList: chatList, OffsetOrder: offsetOrder, Limit: 100, OffsetChatId: offsetChatId}
		chats, err := tdlibClient.GetChats(chatsRequest)
		if err != nil {
			log.Fatalf("[ERROR] GetChats: %s", err)
		}
		log.Printf("GetChats got page %d with %d chats", page, chats.TotalCount)
		for _, chatId := range chats.ChatIds {
			DLog(fmt.Sprintf("New ChatID %d", chatId))
			chat, err := GetChat(chatId, true)
			if err != nil {
				log.Printf("[ERROR] GetChat id %d: %s", chatId, err)

				continue
			}
			if len(chat.Positions) == 0 {
				log.Printf("Cannot load chat without position %d: %s", chatId, GetChatName(chatId))

				continue
			}
			offsetChatId = chat.Id
			offsetOrder = chat.Positions[0].Order

			fullList = append(fullList, chat)
		}

		if len(chats.ChatIds) == 0 {
			log.Printf("Reached end of the list")

			break
		}
		time.Sleep(1 * time.Second)
		page++
		log.Println()
	}

	return fullList
}

func checkSkippedChat(chatId string) bool {
	if _, ok := ignoreLists.IgnoreAuthorIds[chatId]; ok {

		return true
	}
	if _, ok := ignoreLists.IgnoreChatIds[chatId]; ok {

		return true
	}

	return false
}

func checkSkippedSenderBySavedMessage(chatId int64, messageId int64) bool {
	savedMessage, err := FindUpdateNewMessage(chatId, messageId)
	if err != nil {

		return false
	}

	if checkSkippedChat(strconv.FormatInt(GetChatIdBySender(savedMessage.Message.Sender), 10)) {

		return true
	}

	return false
}

func checkChatFilter(chatId int64) bool {
	for _, filter := range chatFilters {
		for _, chatInFilter := range filter.IncludedChats {
			if chatInFilter == chatId && ignoreLists.IgnoreFolders[filter.Title] {
				//log.Printf("Skip chat %d because it's in skipped folder %s", chatId, filter.Title)

				return true
			}
		}
	}

	return false
}

func loadOptionsList() {
	config.UnmarshalJsonFile("tdlib_options.json", &tdlibOptions)
}