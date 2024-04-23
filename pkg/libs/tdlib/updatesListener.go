package tdlib

import (
	"encoding/json"
	"github.com/alexbilevskiy/tgWatch/pkg/libs/helpers"
	"github.com/alexbilevskiy/tgWatch/pkg/libs/modules"
	"github.com/zelenin/go-tdlib/client"
	"log"
	"strconv"
	"time"
)

func (t *TdApi) ListenUpdates() {
	listener := t.tdlibClient.GetListener()
	defer listener.Close()

	for update := range listener.Updates {
		switch update.GetClass() {
		case client.ClassUpdate:
			typ := update.GetType()
			switch typ {
			case client.TypeUpdateChatActionBar:
			case client.TypeUpdateSuggestedActions:
			case client.TypeUpdateChatTheme:
			case client.TypeUpdateChatThemes:
			case client.TypeUpdateFavoriteStickers:
			case client.TypeUpdateInstalledStickerSets:
			case client.TypeUpdateRecentStickers:
			case client.TypeUpdateSavedAnimations:
			case client.TypeUpdateTrendingStickerSets:
			case client.TypeUpdateChatBlockList:
			case client.TypeUpdateChatDraftMessage:
			case client.TypeUpdateUserStatus:
			case client.TypeUpdateChatReadInbox:
			case client.TypeUpdateChatReadOutbox:
			case client.TypeUpdateUnreadMessageCount:
			case client.TypeUpdateChatUnreadReactionCount:
			case client.TypeUpdateUnreadChatCount:
			case client.TypeUpdateChatIsMarkedAsUnread:
			case client.TypeUpdateChatUnreadMentionCount:
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
			case client.TypeUpdateMessageUnreadReactions:
			case client.TypeUpdateAnimatedEmojiMessageClicked:
			case client.TypeUpdateScopeNotificationSettings:
			case client.TypeUpdateStickerSet:
			case client.TypeUpdateSavedNotificationSounds:
			case client.TypeUpdateChatOnlineMemberCount:
			case client.TypeUpdateChatIsTranslatable:
			case client.TypeUpdateAutosaveSettings:
			case client.TypeUpdateForumTopicInfo:
			case client.TypeUpdateChatAccentColors:
			case client.TypeUpdateAccentColors:
			case client.TypeUpdateProfileAccentColors:
			case client.TypeUpdateChatBackground:
			case client.TypeUpdateChatActiveStories:
			case client.TypeUpdateStoryListChatCount:
			case client.TypeUpdateChatViewAsTopics:
			case client.TypeUpdateQuickReplyShortcuts:

			case client.TypeUpdateSupergroup:
			case client.TypeUpdateSupergroupFullInfo:
			case client.TypeUpdateBasicGroup:
			case client.TypeUpdateBasicGroupFullInfo:
			case client.TypeUpdateUser:
			case client.TypeUpdateUserFullInfo:
			case client.TypeUpdateChatPhoto:

			case client.TypeUpdateMessageSendSucceeded:
				upd := update.(*client.UpdateMessageSendSucceeded)
				t.sentMessages.Store(upd.OldMessageId, upd.Message)
			case client.TypeUpdateMessageSendFailed:
				upd := update.(*client.UpdateMessageSendFailed)
				//@TODO: also put in t.sentMessages
				log.Printf("failed to send message %d: %s", upd.OldMessageId, upd.Error.Message)

			case client.TypeUpdateMessageInteractionInfo:
				//upd := update.(*client.UpdateMessageInteractionInfo)
				//log.Printf("[%d] received interaction update for message in chat `%s`: %s", acc, GetChatName(acc, upd.ChatId), BuildMessageLink(upd.ChatId, upd.MessageId))

			case client.TypeUpdateChatTitle:
				//upd := update.(*client.UpdateChatTitle)
				//@TODO: where to get old name?
				//fmt.Printf("Renamed chat id:%d to `%s`", upd.ChatId, upd.Title))

			case client.TypeUpdateChatHasProtectedContent:
				upd := update.(*client.UpdateChatHasProtectedContent)
				log.Printf("Chat id:%d `%s` now has protected content: %s", upd.ChatId, t.GetChatName(upd.ChatId), helpers.JsonMarshalStr(upd.HasProtectedContent))

			case client.TypeUpdateNewChat:
				//dont need to cache chat here, because chat info is empty, @see case client.ClassChat below
				//upd := update.(*client.UpdateNewChat)
				//CacheChat(acc, upd.Chat)
				////fmt.Printf("New chat added: %d / %s", upd.Chat.Id, upd.Chat.Title))
				//saveAllChatPositions(acc, upd.Chat.Id, upd.Chat.Positions)

			case client.TypeUpdateConnectionState:
				//upd := update.(*client.UpdateConnectionState)
				//fmt.Printf("Connection state changed: %s", upd.State.ConnectionStateType()))

			case client.TypeUpdateChatAction:
				upd := update.(*client.UpdateChatAction)
				if upd.ChatId < 0 {
					//fmt.Printf("Skipping action in non-user chat %d: %s", upd.ChatId, upd.Action.ChatActionType()))
					break
				}
				localChat, err := t.GetChat(upd.ChatId, false)
				if err == nil {
					if localChat.LastMessage != nil && localChat.LastMessage.Date < int32(time.Now().Unix())-int32((time.Hour*6).Seconds()) {
						//fmt.Printf("User action in chat `%s`: %s", localChat.Title, upd.Action.ChatActionType()))
					} else {
						//fmt.Printf("Skipping action because its from fresh chat %d `%s`: %s\n", upd.ChatId, localChat.Title, upd.Action.ChatActionType()))
					}
				} else {
					//@NOTE: sender could not be "channel" here, because we only log private chats
					//user, err := t.GetUser(GetChatIdBySender(upd.SenderId))
					//userName := "err_name"
					//if err != nil {
					//	fmt.Printf("failed to get user %d: %s\n", user.Id, err)
					//} else {
					//	userName = getUserFullname(user)
					//}
					//fmt.Printf("User action `%s`: %s", userName, upd.Action.ChatActionType()))
				}

			case client.TypeUpdateChatLastMessage:
				upd := update.(*client.UpdateChatLastMessage)
				if len(upd.Positions) == 0 {
					break
				}
				t.db.SaveAllChatPositions(upd.ChatId, upd.Positions)

			case client.TypeUpdateOption:
				upd := update.(*client.UpdateOption)
				if upd.Name != "unix_time" {
					log.Printf("Update option %s: %s", upd.Name, helpers.JsonMarshalStr(upd.Value))
				}

			case client.TypeUpdateChatPosition:
				upd := update.(*client.UpdateChatPosition)
				t.db.SaveChatPosition(upd.ChatId, upd.Position)

			case client.TypeUpdateChatFolders:
				upd := update.(*client.UpdateChatFolders)
				t.SaveChatFilters(upd)

			case client.TypeUpdateChatAddedToList:
				upd := update.(*client.UpdateChatAddedToList)
				t.SaveChatAddedToList(upd)

			case client.TypeUpdateChatRemovedFromList:
				upd := update.(*client.UpdateChatRemovedFromList)
				t.RemoveChatRemovedFromList(upd)

			case client.TypeUpdateDeleteMessages:
				upd := update.(*client.UpdateDeleteMessages)
				if !upd.IsPermanent || upd.FromCache {

					break
				}
				//chatName := GetChatName(acc, upd.ChatId)
				//intLink := fmt.Sprintf("http://%s/h/%d/?ids=%s", config.Config.WebListen, upd.ChatId, ImplodeInt(upd.MessageIds))
				//count := len(upd.MessageIds)
				//fmt.Printf("DELETED %d Messages from chat: %d, `%s`, %s", count, upd.ChatId, chatName, intLink))

			case client.TypeUpdateNewMessage:
				upd := update.(*client.UpdateNewMessage)
				if t.checkSkippedChat(strconv.FormatInt(upd.Message.ChatId, 10)) || t.checkSkippedChat(strconv.FormatInt(GetChatIdBySender(upd.Message.SenderId), 10)) || t.checkChatFilter(upd.Message.ChatId) {

					break
				}
				//senderChatId := GetChatIdBySender(upd.Message.Sender)
				//mongoId := SaveUpdate(t, upd, upd.Message.Date)
				//link := GetLink(tdlibClient, upd.Message.ChatId, upd.Message.Id)
				//chatName := GetChatName(upd.Message.ChatId)
				//intLink := fmt.Sprintf("http://%s/m/%d/%d", config.Config.WebListen, upd.Message.ChatId, upd.Message.Id)
				//log.Printf("[%s] New Message from chat: %d, `%s`, %s, %s", mongoId, upd.Message.ChatId, chatName, link, intLink)
				if upd.Message.Content.MessageContentType() == client.TypeMessageChatAddMembers ||
					upd.Message.Content.MessageContentType() == client.TypeMessageChatJoinByLink {
					t.MarkJoinAsRead(upd.Message.ChatId, upd.Message.Id)
				}

				modules.CustomNewMessageRoutine(t.dbData.Id, t.tdlibClient, upd)

			case client.TypeUpdateMessageEdited:
				upd := update.(*client.UpdateMessageEdited)
				if t.checkSkippedChat(strconv.FormatInt(upd.ChatId, 10)) || t.checkChatFilter(upd.ChatId) {

					break
				}
				if upd.ReplyMarkup != nil {
					//messages with buttons - reactions, likes etc
					break
				}

				//mongoId := SaveUpdate(t, upd, upd.EditDate)
				//link := GetLink(tdlibClient, upd.ChatId, upd.MessageId)
				//chatName := GetChatName(upd.ChatId)
				//intLink := fmt.Sprintf("http://%s/m/%d/%d", config.Config.WebListen, upd.ChatId, upd.MessageId)
				//log.Printf("[%s] EDITED msg! Chat: %d, msg %d, `%s`, %s, %s", mongoId, upd.ChatId, upd.MessageId, chatName, link, intLink)

			case client.TypeUpdateMessageContent:
				upd := update.(*client.UpdateMessageContent)
				if t.checkSkippedChat(strconv.FormatInt(upd.ChatId, 10)) || t.checkChatFilter(upd.ChatId) {

					break
				}
				if upd.NewContent.MessageContentType() == client.TypeMessagePoll {
					//dont save "poll" updates - that's just counters, users cannot update polls manually
					break
				}
				//link := t.GetLink(upd.ChatId, upd.MessageId)
				//chatName := t.GetChatName(upd.ChatId)
				//intLink := fmt.Sprintf("http://%s/m/%d/%d", config.Config.WebListen, upd.ChatId, upd.MessageId)
				//fmt.Printf("EDITED content! Chat: %d, msg %d, %s, %s, %s", upd.ChatId, upd.MessageId, chatName, link, intLink))

				modules.CustomMessageContentRoutine(t.dbData.Id, t.tdlibClient, upd)

			case client.TypeUpdateFile:
				upd := update.(*client.UpdateFile)
				if upd.File.Local.IsDownloadingActive {
					//fmt.Printf("File downloading: %d/%d bytes", upd.File.Local.DownloadedSize, upd.File.ExpectedSize))
				} else {
					//fmt.Printf("File downloaded: %d bytes, path: %s", upd.File.Local.DownloadedSize, upd.File.Local.Path))
				}

			case client.TypeUpdateChatMessageAutoDeleteTime:
				upd := update.(*client.UpdateChatMessageAutoDeleteTime)
				chatName := t.GetChatName(upd.ChatId)
				log.Printf("Message auto-delete time updated for chat `%s` %d: %ds", chatName, upd.ChatId, upd.MessageAutoDeleteTime)

			case client.TypeUpdateChatAvailableReactions:
				//upd := update.(*client.UpdateChatAvailableReactions)
				//chatName := t.GetChatName(upd.ChatId)
				//fmt.Printf("Available reactions updated for chat `%s` %d: %s", chatName, upd.ChatId, JsonMarshalStr(upd.AvailableReactions)))

			default:
				j, _ := json.Marshal(update)
				log.Printf("Unknown update %s : %s", typ, string(j))
			}

		case client.ClassOk:
		case client.ClassError:
		case client.ClassUser:
		case client.ClassChat:
			upd := update.(*client.Chat)
			t.cacheChat(upd)
		case client.ClassSupergroup:
		case client.ClassChats:
		case client.ClassMessageLink:
		case client.ClassFile:
		case client.ClassChatFolder:
		case client.ClassOptionValue:
		case client.ClassChatMember:
		case client.ClassSessions:
		case client.ClassMessage:
		case client.ClassMessages:
		case client.ClassInternalLinkType:
		case client.ClassChatInviteLinkInfo:
		case client.ClassMessageLinkInfo:
		case client.ClassStickers:

		default:
			log.Printf("WAAAT? update who??? %s, %v", update.GetClass(), update)
		}
	}
}
