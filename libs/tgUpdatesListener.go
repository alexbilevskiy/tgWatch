package libs

import (
	"encoding/json"
	"fmt"
	"go-tdlib/client"
	"log"
	"strconv"
	"tgWatch/config"
)

func ListenUpdates()  {
	listener := tdlibClient.GetListener()
	defer listener.Close()

	for update := range listener.Updates {
		switch update.GetClass() {
		case client.ClassUpdate:
			t := update.GetType()
			switch t {
			case client.TypeUpdateChatActionBar:
			case client.TypeUpdateFavoriteStickers:
			case client.TypeUpdateInstalledStickerSets:
			case client.TypeUpdateRecentStickers:
			case client.TypeUpdateSavedAnimations:
			case client.TypeUpdateTrendingStickerSets:
			case client.TypeUpdateChatIsBlocked:
			case client.TypeUpdateChatDraftMessage:
			case client.TypeUpdateUserStatus:
			case client.TypeUpdateChatReadInbox:
			case client.TypeUpdateChatReadOutbox:
			case client.TypeUpdateUnreadMessageCount:
			case client.TypeUpdateUnreadChatCount:
			case client.TypeUpdateChatIsMarkedAsUnread:
			case client.TypeUpdateChatUnreadMentionCount:
			case client.TypeUpdateMessageInteractionInfo:
			case client.TypeUpdateChatReplyMarkup:
			case client.TypeUpdateChatPermissions:
			case client.TypeUpdateChatNotificationSettings:
			case client.TypeUpdateMessageMentionRead:
			case client.TypeUpdateMessageIsPinned:
			case client.TypeUpdateChatHasScheduledMessages:
			case client.TypeUpdateHavePendingNotifications:
			case client.TypeUpdateCall:
			case client.TypeUpdateMessageContentOpened:
			case client.TypeUpdateUserPrivacySettingRules:

			case client.TypeUpdateSupergroup:
			case client.TypeUpdateSupergroupFullInfo:
			case client.TypeUpdateBasicGroup:
			case client.TypeUpdateBasicGroupFullInfo:
			case client.TypeUpdateUser:
			case client.TypeUpdateUserFullInfo:
			case client.TypeUpdateChatPhoto:
				//break
				//golang WTF? U dont need break??

			case client.TypeUpdateChatTitle:
				upd := update.(*client.UpdateChatTitle)
				log.Printf("Renamed chat id:%d to `%s`", upd.ChatId, upd.Title)

				break
			case client.TypeUpdateNewChat:
				upd := update.(*client.UpdateNewChat)
				localChats[upd.Chat.Id] = upd.Chat
				DLog(fmt.Sprintf("New chat added: %d / %s", upd.Chat.Id, upd.Chat.Title))
				saveAllChatPositions(upd.Chat.Id, upd.Chat.Positions)

				break
			case client.TypeUpdateConnectionState:
				upd := update.(*client.UpdateConnectionState)
				log.Printf("Connection state changed: %s", upd.State.ConnectionStateType())

				break
			case client.TypeUpdateUserChatAction:
				upd := update.(*client.UpdateUserChatAction)
				if upd.ChatId < 0 {
					DLog(fmt.Sprintf("Skipping action in non-user chat %d: %s", upd.ChatId, upd.Action.ChatActionType()))
					break
				}
				user, err := GetUser(upd.UserId)
				userName := "err_name"
				if err != nil {
					fmt.Printf("failed to get user %d: %s", upd.UserId, err)
				} else {
					userName = getUserFullname(user)
				}
				log.Printf("User action `%s`: %s", userName, upd.Action.ChatActionType())

				break
			case client.TypeUpdateChatLastMessage:
				upd := update.(*client.UpdateChatLastMessage)
				if len(upd.Positions) == 0 {
					break
				}
				saveAllChatPositions(upd.ChatId, upd.Positions)

				break
			case client.TypeUpdateOption:
				upd := update.(*client.UpdateOption)
				log.Printf("Update option %s: %s", upd.Name, JsonMarshalStr(upd.Value))

				break
			case client.TypeUpdateChatPosition:
				upd := update.(*client.UpdateChatPosition)
				saveChatPosition(upd.ChatId, upd.Position)

				break
			case client.TypeUpdateChatFilters:
				upd := update.(*client.UpdateChatFilters)
				SaveChatFilters(upd)

				break

			case client.TypeUpdateDeleteMessages:
				upd := update.(*client.UpdateDeleteMessages)
				if !upd.IsPermanent || upd.FromCache {

					break
				}
				if checkSkippedChat(strconv.FormatInt(upd.ChatId, 10)) || checkChatFilter(upd.ChatId) {

					break
				}
				MarkAsDeleted(upd.ChatId, upd.MessageIds)

				skipUpdate := 0
				realUpdates := make([]int64, 0)
				for _, messageId := range upd.MessageIds {
					savedMessage, err := FindUpdateNewMessage(upd.ChatId, messageId)
					if err != nil {
						realUpdates = append(realUpdates, messageId)

						continue
					}
					if checkSkippedChat(strconv.FormatInt(GetChatIdBySender(savedMessage.Message.Sender), 10)) {
						DLog(fmt.Sprintf("Skip deleted message %d from sender %d, `%s`", messageId, GetChatIdBySender(savedMessage.Message.Sender), GetSenderName(savedMessage.Message.Sender)))
						skipUpdate++

						continue
					}
					if savedMessage.Message.Content == nil {
						log.Printf("Skip deleted message %d with unknown content from %d", messageId, GetChatIdBySender(savedMessage.Message.Sender))

						continue
					}
					if savedMessage.Message.Content.MessageContentType() == "messageChatAddMembers" {
						DLog(fmt.Sprintf("Skip deleted message %d (chat join of user %d)", messageId, GetChatIdBySender(savedMessage.Message.Sender)))
						skipUpdate++

						continue
					}
					realUpdates = append(realUpdates, messageId)
				}
				if len(realUpdates) <= 0 {

					break
				}
				upd.MessageIds = realUpdates
				mongoId := SaveUpdate(t, upd, 0)

				chatName := GetChatName(upd.ChatId)
				intLink := fmt.Sprintf("http://%s/h/%d/?ids=%s", config.Config.WebListen, upd.ChatId, ImplodeInt(upd.MessageIds))
				count := len(upd.MessageIds)
				DLog(fmt.Sprintf("[%s] DELETED %d Messages from chat: %d, `%s`, %s", mongoId, count, upd.ChatId, chatName, intLink))

				break

			case client.TypeUpdateNewMessage:
				upd := update.(*client.UpdateNewMessage)
				if checkSkippedChat(strconv.FormatInt(upd.Message.ChatId, 10)) || checkSkippedChat(strconv.FormatInt(GetChatIdBySender(upd.Message.Sender), 10)) || checkChatFilter(upd.Message.ChatId) {

					break
				}
				//senderChatId := GetChatIdBySender(upd.Message.Sender)
				SaveUpdate(t, upd, upd.Message.Date)
				//mongoId := SaveUpdate(t, upd, upd.Message.Date)
				//link := GetLink(tdlibClient, upd.Message.ChatId, upd.Message.Id)
				//chatName := GetChatName(upd.Message.ChatId)
				//intLink := fmt.Sprintf("http://%s/m/%d/%d", config.Config.WebListen, upd.Message.ChatId, upd.Message.Id)
				//log.Printf("[%s] New Message from chat: %d, `%s`, %s, %s", mongoId, upd.Message.ChatId, chatName, link, intLink)

				break
			case client.TypeUpdateMessageEdited:
				upd := update.(*client.UpdateMessageEdited)
				if checkSkippedChat(strconv.FormatInt(upd.ChatId, 10)) || checkChatFilter(upd.ChatId) {

					break
				}
				if upd.ReplyMarkup != nil {
					//messages with buttons - reactions, likes etc
					break
				}
				if checkSkippedSenderBySavedMessage(upd.ChatId, upd.MessageId) {

					break
				}

				SaveUpdate(t, upd, upd.EditDate)
				//mongoId := SaveUpdate(t, upd, upd.EditDate)
				//link := GetLink(tdlibClient, upd.ChatId, upd.MessageId)
				//chatName := GetChatName(upd.ChatId)
				//intLink := fmt.Sprintf("http://%s/m/%d/%d", config.Config.WebListen, upd.ChatId, upd.MessageId)
				//log.Printf("[%s] EDITED msg! Chat: %d, msg %d, `%s`, %s, %s", mongoId, upd.ChatId, upd.MessageId, chatName, link, intLink)

				break
			case client.TypeUpdateMessageContent:
				upd := update.(*client.UpdateMessageContent)
				//@TODO: find message in DB and check sender, maybe he is ignored
				if checkSkippedChat(strconv.FormatInt(upd.ChatId, 10)) || checkChatFilter(upd.ChatId) {

					break
				}
				if upd.NewContent.MessageContentType() == "messagePoll" {
					//dont save "poll" updates - that's just counters, users cannot update polls manually
					break
				}
				if checkSkippedSenderBySavedMessage(upd.ChatId, upd.MessageId) {

					break
				}

				mongoId := SaveUpdate(t, upd, 0)

				link := GetLink(upd.ChatId, upd.MessageId)
				chatName := GetChatName(upd.ChatId)
				intLink := fmt.Sprintf("http://%s/m/%d/%d", config.Config.WebListen, upd.ChatId, upd.MessageId)
				DLog(fmt.Sprintf("[%s] EDITED content! Chat: %d, msg %d, %s, %s, %s", mongoId, upd.ChatId, upd.MessageId, chatName, link, intLink))

				break
			case client.TypeUpdateFile:
				upd := update.(*client.UpdateFile)
				if upd.File.Local.IsDownloadingActive {
					DLog(fmt.Sprintf("File downloading: %d/%d bytes", upd.File.Local.DownloadedSize, upd.File.ExpectedSize))
				} else {
					DLog(fmt.Sprintf("File downloaded: %d bytes, path: %s", upd.File.Local.DownloadedSize, upd.File.Local.Path))
				}

				break
			default:
				j, _ := json.Marshal(update)
				log.Printf("Unknown update %s : %s", t, string(j))
			}
			break
		case client.ClassOk:
		case client.ClassError:
		case client.ClassUser:
		case client.ClassChat:
		case client.ClassSupergroup:
		case client.ClassChats:
		case client.ClassMessageLink:
		case client.ClassFile:
		case client.ClassChatFilter:
		case client.ClassOptionValue:
		case client.ClassChatMember:
		case client.ClassSessions:
			break
		default:
			log.Printf("WAAAT? update who??? %s, %v", update.GetClass(), update)
		}
	}
}
