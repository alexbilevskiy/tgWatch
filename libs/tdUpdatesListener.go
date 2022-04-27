package libs

import (
	"encoding/json"
	"fmt"
	"go-tdlib/client"
	"log"
	"strconv"
	"tgWatch/config"
	"time"
)

func ListenUpdates(acc int64) {
	listener := tdlibClient[acc].GetListener()
	defer listener.Close()

	for update := range listener.Updates {
		switch update.GetClass() {
		case client.ClassUpdate:
			t := update.GetType()
			switch t {
			case client.TypeUpdateChatActionBar:
			case client.TypeUpdateSuggestedActions:
			case client.TypeUpdateChatThemes:
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
			case client.TypeUpdateGroupCall:
			case client.TypeUpdateChatVideoChat:
			case client.TypeUpdateChatMessageSender:
			case client.TypeUpdateReactions:
			case client.TypeUpdateMessageUnreadReactions:

			case client.TypeUpdateSupergroup:
			case client.TypeUpdateSupergroupFullInfo:
			case client.TypeUpdateBasicGroup:
			case client.TypeUpdateBasicGroupFullInfo:
			case client.TypeUpdateUser:
			case client.TypeUpdateUserFullInfo:
			case client.TypeUpdateChatPhoto:
			case client.TypeUpdateMessageSendSucceeded:
				//break
				//golang WTF? U dont need break??

			case client.TypeUpdateChatTitle:
				upd := update.(*client.UpdateChatTitle)
				//@TODO: where to get old name?
				DLog(fmt.Sprintf("Renamed chat id:%d to `%s`", upd.ChatId, upd.Title))

			case client.TypeUpdateChatHasProtectedContent:
				upd := update.(*client.UpdateChatHasProtectedContent)
				log.Printf("Chat id:%d `%s` now has protected content: %s", upd.ChatId, GetChatName(acc, upd.ChatId), JsonMarshalStr(upd.HasProtectedContent))

			case client.TypeUpdateNewChat:
				//dont need to cache chat here, because chat info is empty, @see case client.ClassChat below
				//upd := update.(*client.UpdateNewChat)
				//CacheChat(acc, upd.Chat)
				//DLog(fmt.Sprintf("New chat added: %d / %s", upd.Chat.Id, upd.Chat.Title))
				//saveAllChatPositions(acc, upd.Chat.Id, upd.Chat.Positions)

				break
			case client.TypeUpdateConnectionState:
				upd := update.(*client.UpdateConnectionState)
				DLog(fmt.Sprintf("Connection state changed: %s", upd.State.ConnectionStateType()))

				break
			case client.TypeUpdateChatAction:
				upd := update.(*client.UpdateChatAction)
				if upd.ChatId < 0 {
					DLog(fmt.Sprintf("Skipping action in non-user chat %d: %s", upd.ChatId, upd.Action.ChatActionType()))
					break
				}
				localChat, err := GetChat(acc, upd.ChatId, false)
				if err == nil {
					if localChat.LastMessage != nil && localChat.LastMessage.Date < int32(time.Now().Unix())-int32((time.Hour*6).Seconds()) {
						DLog(fmt.Sprintf("User action in chat `%s`: %s", localChat.Title, upd.Action.ChatActionType()))
					} else {
						DLog(fmt.Sprintf("Skipping action because its from fresh chat %d `%s`: %s\n", upd.ChatId, localChat.Title, upd.Action.ChatActionType()))
					}
				} else {
					//@NOTE: sender could not be "channel" here, because we only log private chats
					user, err := GetUser(acc, GetChatIdBySender(upd.SenderId))
					userName := "err_name"
					if err != nil {
						fmt.Printf("failed to get user %d: %s\n", user.Id, err)
					} else {
						userName = getUserFullname(user)
					}
					DLog(fmt.Sprintf("User action `%s`: %s", userName, upd.Action.ChatActionType()))
				}

				break
			case client.TypeUpdateChatLastMessage:
				upd := update.(*client.UpdateChatLastMessage)
				if len(upd.Positions) == 0 {
					break
				}
				saveAllChatPositions(acc, upd.ChatId, upd.Positions)

				break
			case client.TypeUpdateOption:
				upd := update.(*client.UpdateOption)
				if upd.Name != "unix_time" {
					log.Printf("Update option %s: %s", upd.Name, JsonMarshalStr(upd.Value))
				}

				break
			case client.TypeUpdateChatPosition:
				upd := update.(*client.UpdateChatPosition)
				saveChatPosition(acc, upd.ChatId, upd.Position)

				break
			case client.TypeUpdateChatFilters:
				upd := update.(*client.UpdateChatFilters)
				SaveChatFilters(acc, upd)

				break

			case client.TypeUpdateDeleteMessages:
				upd := update.(*client.UpdateDeleteMessages)
				if !upd.IsPermanent || upd.FromCache {

					break
				}
				if checkSkippedChat(acc, strconv.FormatInt(upd.ChatId, 10)) || checkChatFilter(acc, upd.ChatId) {

					break
				}
				MarkAsDeleted(acc, upd.ChatId, upd.MessageIds)

				skipUpdate := 0
				realUpdates := make([]int64, 0)
				for _, messageId := range upd.MessageIds {
					savedMessage, err := FindUpdateNewMessage(acc, upd.ChatId, messageId)
					if err != nil {
						realUpdates = append(realUpdates, messageId)

						continue
					}
					if checkSkippedChat(acc, strconv.FormatInt(GetChatIdBySender(savedMessage.Message.SenderId), 10)) {
						DLog(fmt.Sprintf("Skip deleted message %d from sender %d, `%s`", messageId, GetChatIdBySender(savedMessage.Message.SenderId), GetSenderName(acc, savedMessage.Message.SenderId)))
						skipUpdate++

						continue
					}
					if savedMessage.Message.Content == nil {
						log.Printf("Skip deleted message %d with unknown content from %d", messageId, GetChatIdBySender(savedMessage.Message.SenderId))

						continue
					}
					if savedMessage.Message.Content.MessageContentType() == client.TypeMessageChatAddMembers {
						DLog(fmt.Sprintf("Skip deleted message %d (chat join of user %d)", messageId, GetChatIdBySender(savedMessage.Message.SenderId)))
						skipUpdate++

						continue
					}
					realUpdates = append(realUpdates, messageId)
				}
				if len(realUpdates) <= 0 {

					break
				}
				upd.MessageIds = realUpdates
				mongoId := SaveUpdate(acc, t, upd, 0)

				chatName := GetChatName(acc, upd.ChatId)
				intLink := fmt.Sprintf("http://%s/h/%d/?ids=%s", config.Config.WebListen, upd.ChatId, ImplodeInt(upd.MessageIds))
				count := len(upd.MessageIds)
				DLog(fmt.Sprintf("[%s] DELETED %d Messages from chat: %d, `%s`, %s", mongoId, count, upd.ChatId, chatName, intLink))

				break

			case client.TypeUpdateNewMessage:
				upd := update.(*client.UpdateNewMessage)
				if checkSkippedChat(acc, strconv.FormatInt(upd.Message.ChatId, 10)) || checkSkippedChat(acc, strconv.FormatInt(GetChatIdBySender(upd.Message.SenderId), 10)) || checkChatFilter(acc, upd.Message.ChatId) {

					break
				}
				//senderChatId := GetChatIdBySender(upd.Message.Sender)
				SaveUpdate(acc, t, upd, upd.Message.Date)
				//mongoId := SaveUpdate(t, upd, upd.Message.Date)
				//link := GetLink(tdlibClient, upd.Message.ChatId, upd.Message.Id)
				//chatName := GetChatName(upd.Message.ChatId)
				//intLink := fmt.Sprintf("http://%s/m/%d/%d", config.Config.WebListen, upd.Message.ChatId, upd.Message.Id)
				//log.Printf("[%s] New Message from chat: %d, `%s`, %s, %s", mongoId, upd.Message.ChatId, chatName, link, intLink)
				if upd.Message.Content.MessageContentType() == client.TypeMessageChatAddMembers ||
					upd.Message.Content.MessageContentType() == client.TypeMessageChatJoinByLink {
					MarkJoinAsRead(acc, upd.Message.ChatId, upd.Message.Id)
				}

				//"saved messages"
				if upd.Message.ChatId == acc &&
					upd.Message.Content.MessageContentType() == client.TypeMessageVoiceNote {
					ct := upd.Message.Content.(*client.MessageVoiceNote)
					text, err := RecognizeByFileId(acc, ct.VoiceNote.Voice.Remote.Id)
					if err != nil {
						text = "error: " + err.Error()
					} else {
						text = "recognized: " + text
					}
					SendMessage(acc, text, upd.Message.ChatId, &upd.Message.Id)
				}

				break
			case client.TypeUpdateMessageEdited:
				upd := update.(*client.UpdateMessageEdited)
				if checkSkippedChat(acc, strconv.FormatInt(upd.ChatId, 10)) || checkChatFilter(acc, upd.ChatId) {

					break
				}
				if upd.ReplyMarkup != nil {
					//messages with buttons - reactions, likes etc
					break
				}
				if checkSkippedSenderBySavedMessage(acc, upd.ChatId, upd.MessageId) {

					break
				}

				SaveUpdate(acc, t, upd, upd.EditDate)
				//mongoId := SaveUpdate(t, upd, upd.EditDate)
				//link := GetLink(tdlibClient, upd.ChatId, upd.MessageId)
				//chatName := GetChatName(upd.ChatId)
				//intLink := fmt.Sprintf("http://%s/m/%d/%d", config.Config.WebListen, upd.ChatId, upd.MessageId)
				//log.Printf("[%s] EDITED msg! Chat: %d, msg %d, `%s`, %s, %s", mongoId, upd.ChatId, upd.MessageId, chatName, link, intLink)

				break
			case client.TypeUpdateMessageContent:
				upd := update.(*client.UpdateMessageContent)
				//@TODO: find message in DB and check sender, maybe he is ignored
				if checkSkippedChat(acc, strconv.FormatInt(upd.ChatId, 10)) || checkChatFilter(acc, upd.ChatId) {

					break
				}
				if upd.NewContent.MessageContentType() == client.TypeMessagePoll {
					//dont save "poll" updates - that's just counters, users cannot update polls manually
					break
				}
				if checkSkippedSenderBySavedMessage(acc, upd.ChatId, upd.MessageId) {

					break
				}

				mongoId := SaveUpdate(acc, t, upd, 0)

				link := GetLink(acc, upd.ChatId, upd.MessageId)
				chatName := GetChatName(acc, upd.ChatId)
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
			case client.TypeUpdateChatMessageTtl:
				upd := update.(*client.UpdateChatMessageTtl)
				chatName := GetChatName(acc, upd.ChatId)
				log.Printf("Message TTL updated for chat `%s` %d: %ds", chatName, upd.ChatId, upd.MessageTtl)
			case client.TypeUpdateChatAvailableReactions:
				upd := update.(*client.UpdateChatAvailableReactions)
				chatName := GetChatName(acc, upd.ChatId)
				DLog(fmt.Sprintf("Available reactions updated for chat `%s` %d: %s", chatName, upd.ChatId, JsonMarshalStr(upd.AvailableReactions)))
			default:
				j, _ := json.Marshal(update)
				log.Printf("Unknown update %s : %s", t, string(j))
			}
			break
		case client.ClassOk:
		case client.ClassError:
		case client.ClassUser:
		case client.ClassChat:
			upd := update.(*client.Chat)
			CacheChat(acc, upd)
		case client.ClassSupergroup:
		case client.ClassChats:
		case client.ClassMessageLink:
		case client.ClassFile:
		case client.ClassChatFilter:
		case client.ClassOptionValue:
		case client.ClassChatMember:
		case client.ClassSessions:
		case client.ClassMessage:
		case client.ClassInternalLinkType:
		case client.ClassChatInviteLinkInfo:
		case client.ClassMessageLinkInfo:
			break
		default:
			log.Printf("WAAAT? update who??? %s, %v", update.GetClass(), update)
		}
	}
}
