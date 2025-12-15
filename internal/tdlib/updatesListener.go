package tdlib

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/alexbilevskiy/tgWatch/internal/helpers"
	"github.com/alexbilevskiy/tgWatch/internal/modules"
	"github.com/zelenin/go-tdlib/client"
)

func (t *TdApi) UpdatesCallback(ctx context.Context, update client.Type) {
	switch update.GetType() {
	case client.TypeUpdate:
		typ := update.GetConstructor()
		switch typ {
		case client.ConstructorUpdateAuthorizationState:
			//@todo: do we need to do anything?

		case client.ConstructorUpdateChatActionBar:
		case client.ConstructorUpdateSuggestedActions:
		case client.ConstructorUpdateChatTheme:
		case client.ConstructorUpdateChatThemes:
		case client.ConstructorUpdateFavoriteStickers:
		case client.ConstructorUpdateInstalledStickerSets:
		case client.ConstructorUpdateRecentStickers:
		case client.ConstructorUpdateSavedAnimations:
		case client.ConstructorUpdateTrendingStickerSets:
		case client.ConstructorUpdateChatBlockList:
		case client.ConstructorUpdateChatDraftMessage:
		case client.ConstructorUpdateUserStatus:
		case client.ConstructorUpdateChatReadInbox:
		case client.ConstructorUpdateChatReadOutbox:
		case client.ConstructorUpdateUnreadMessageCount:
		case client.ConstructorUpdateChatUnreadReactionCount:
		case client.ConstructorUpdateUnreadChatCount:
		case client.ConstructorUpdateChatIsMarkedAsUnread:
		case client.ConstructorUpdateChatUnreadMentionCount:
		case client.ConstructorUpdateChatReplyMarkup:
		case client.ConstructorUpdateChatPermissions:
		case client.ConstructorUpdateChatNotificationSettings:
		case client.ConstructorUpdateMessageMentionRead:
		case client.ConstructorUpdateMessageIsPinned:
		case client.ConstructorUpdateChatHasScheduledMessages:
		case client.ConstructorUpdateHavePendingNotifications:
		case client.ConstructorUpdateCall:
		case client.ConstructorUpdateMessageContentOpened:
		case client.ConstructorUpdateUserPrivacySettingRules:
		case client.ConstructorUpdateGroupCall:
		case client.ConstructorUpdateChatVideoChat:
		case client.ConstructorUpdateChatMessageSender:
		case client.ConstructorUpdateMessageUnreadReactions:
		case client.ConstructorUpdateAnimatedEmojiMessageClicked:
		case client.ConstructorUpdateScopeNotificationSettings:
		case client.ConstructorUpdateStickerSet:
		case client.ConstructorUpdateSavedNotificationSounds:
		case client.ConstructorUpdateChatOnlineMemberCount:
		case client.ConstructorUpdateChatIsTranslatable:
		case client.ConstructorUpdateAutosaveSettings:
		case client.ConstructorUpdateForumTopicInfo:
		case client.ConstructorUpdateChatAccentColors:
		case client.ConstructorUpdateAccentColors:
		case client.ConstructorUpdateProfileAccentColors:
		case client.ConstructorUpdateChatBackground:
		case client.ConstructorUpdateChatActiveStories:
		case client.ConstructorUpdateStoryListChatCount:
		case client.ConstructorUpdateChatViewAsTopics:
		case client.ConstructorUpdateQuickReplyShortcuts:
		case client.ConstructorUpdateAvailableMessageEffects:
		case client.ConstructorUpdateDefaultReactionType:
		case client.ConstructorUpdateSavedMessagesTopic:
		case client.ConstructorUpdateSpeechRecognitionTrial:
		case client.ConstructorUpdateAnimationSearchParameters:
		case client.ConstructorUpdateAttachmentMenuBots:
		case client.ConstructorUpdateDefaultBackground:
		case client.ConstructorUpdateFileDownloads:
		case client.ConstructorUpdateFileDownload:
		case client.ConstructorUpdateDiceEmojis:
		case client.ConstructorUpdateActiveEmojiReactions:
		case client.ConstructorUpdateDefaultPaidReactionType:
		case client.ConstructorUpdateOwnedStarCount:
		case client.ConstructorUpdateReactionNotificationSettings:
		case client.ConstructorUpdateStoryStealthMode:

		case client.ConstructorUpdateSupergroup:
		case client.ConstructorUpdateSupergroupFullInfo:
		case client.ConstructorUpdateBasicGroup:
		case client.ConstructorUpdateBasicGroupFullInfo:
		case client.ConstructorUpdateUser:
		case client.ConstructorUpdateUserFullInfo:
		case client.ConstructorUpdateChatPhoto:

		case client.ConstructorUpdateMessageSendSucceeded:
			upd := update.(*client.UpdateMessageSendSucceeded)
			t.sentMessages.Store(upd.OldMessageId, upd.Message)
		case client.ConstructorUpdateMessageSendFailed:
			upd := update.(*client.UpdateMessageSendFailed)
			//@TODO: also put in t.sentMessages
			log.Printf("failed to send message %d: %s", upd.OldMessageId, upd.Error.Message)

		case client.ConstructorUpdateMessageInteractionInfo:
			//upd := update.(*client.UpdateMessageInteractionInfo)
			//log.Printf("[%d] received interaction update for message in chat `%s`: %s", acc, GetChatName(acc, upd.ChatId), BuildMessageLink(upd.ChatId, upd.MessageId))

		case client.ConstructorUpdateChatTitle:
			//upd := update.(*client.UpdateChatTitle)
			//@TODO: where to get old name?
			//fmt.Printf("Renamed chat id:%d to `%s`", upd.ChatId, upd.Title))

		case client.ConstructorUpdateChatHasProtectedContent:
			upd := update.(*client.UpdateChatHasProtectedContent)
			log.Printf("Chat id:%d `%s` now has protected content: %s", upd.ChatId, t.GetChatName(ctx, upd.ChatId), helpers.JsonMarshalStr(upd.HasProtectedContent))

		case client.ConstructorUpdateNewChat:
			//dont need to cache chat here, because chat info is empty, @see case client.ClassChat below
			//upd := update.(*client.UpdateNewChat)
			//CacheChat(acc, upd.Chat)
			////fmt.Printf("New chat added: %d / %s", upd.Chat.Id, upd.Chat.Title))
			//saveAllChatPositions(acc, upd.Chat.Id, upd.Chat.Positions)

		case client.ConstructorUpdateConnectionState:
			//upd := update.(*client.UpdateConnectionState)
			//fmt.Printf("Connection state changed: %s", upd.State.ConnectionStateType()))

		case client.ConstructorUpdateChatAction:
			upd := update.(*client.UpdateChatAction)
			if upd.ChatId < 0 {
				//fmt.Printf("Skipping action in non-user chat %d: %s", upd.ChatId, upd.Action.ChatActionType()))
				break
			}
			localChat, err := t.GetChat(ctx, upd.ChatId, false)
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

		case client.ConstructorUpdateChatLastMessage:
			upd := update.(*client.UpdateChatLastMessage)
			if len(upd.Positions) == 0 {
				break
			}
			t.db.SaveAllChatPositions(ctx, upd.ChatId, upd.Positions)

		case client.ConstructorUpdateOption:
			upd := update.(*client.UpdateOption)
			if upd.Name != "unix_time" {
				//log.Printf("Update option %s: %s", upd.Name, helpers.JsonMarshalStr(upd.Value))
			}

		case client.ConstructorUpdateChatPosition:
			upd := update.(*client.UpdateChatPosition)
			t.db.SaveChatPosition(ctx, upd.ChatId, upd.Position)

		case client.ConstructorUpdateChatFolders:
			upd := update.(*client.UpdateChatFolders)
			t.SaveChatFilters(ctx, upd)

		case client.ConstructorUpdateChatAddedToList:
			upd := update.(*client.UpdateChatAddedToList)
			t.SaveChatAddedToList(ctx, upd)

		case client.ConstructorUpdateChatRemovedFromList:
			upd := update.(*client.UpdateChatRemovedFromList)
			t.RemoveChatRemovedFromList(ctx, upd)

		case client.ConstructorUpdateDeleteMessages:
			upd := update.(*client.UpdateDeleteMessages)
			if !upd.IsPermanent || upd.FromCache {

				break
			}
			//chatName := GetChatName(acc, upd.ChatId)
			//intLink := fmt.Sprintf("http://%s/h/%d/?ids=%s", config.Config.WebListen, upd.ChatId, ImplodeInt(upd.MessageIds))
			//count := len(upd.MessageIds)
			//fmt.Printf("DELETED %d Messages from chat: %d, `%s`, %s", count, upd.ChatId, chatName, intLink))

		case client.ConstructorUpdateNewMessage:
			upd := update.(*client.UpdateNewMessage)
			if upd.Message.Content.MessageContentConstructor() == client.ConstructorMessageChatAddMembers ||
				upd.Message.Content.MessageContentConstructor() == client.ConstructorMessageChatJoinByLink {
				t.MarkJoinAsRead(ctx, upd.Message.ChatId, upd.Message.Id)
			}

			modules.CustomNewMessageRoutine(ctx, t.dbData.Id, t.tdlibClient, upd)

		case client.ConstructorUpdateMessageEdited:
			upd := update.(*client.UpdateMessageEdited)
			if upd.ReplyMarkup != nil {
				//messages with buttons - reactions, likes etc
				break
			}

		case client.ConstructorUpdateMessageContent:
			upd := update.(*client.UpdateMessageContent)
			if upd.NewContent.MessageContentConstructor() == client.ConstructorMessagePoll {
				//dont save "poll" updates - that's just counters, users cannot update polls manually
				break
			}

			modules.CustomMessageContentRoutine(ctx, t.dbData.Id, t.tdlibClient, upd)

		case client.ConstructorUpdateFile:
			upd := update.(*client.UpdateFile)
			if upd.File.Local.IsDownloadingActive {
				//fmt.Printf("File downloading: %d/%d bytes", upd.File.Local.DownloadedSize, upd.File.ExpectedSize))
			} else {
				//fmt.Printf("File downloaded: %d bytes, path: %s", upd.File.Local.DownloadedSize, upd.File.Local.Path))
			}

		case client.ConstructorUpdateChatMessageAutoDeleteTime:
			upd := update.(*client.UpdateChatMessageAutoDeleteTime)
			chatName := t.GetChatName(ctx, upd.ChatId)
			log.Printf("Message auto-delete time updated for chat `%s` %d: %ds", chatName, upd.ChatId, upd.MessageAutoDeleteTime)

		case client.ConstructorUpdateChatAvailableReactions:
			//upd := update.(*client.UpdateChatAvailableReactions)
			//chatName := t.GetChatName(upd.ChatId)
			//fmt.Printf("Available reactions updated for chat `%s` %d: %s", chatName, upd.ChatId, JsonMarshalStr(upd.AvailableReactions)))

		default:
			j, _ := json.Marshal(update)
			log.Printf("Unknown update %s : %s", typ, string(j))
		}

	case client.TypeOk:
	case client.TypeError:
	case client.TypeUser:
	case client.TypeChat:
		upd := update.(*client.Chat)
		t.cacheChat(upd)
	case client.TypeSupergroup:
	case client.TypeChats:
	case client.TypeMessageLink:
	case client.TypeFile:
	case client.TypeChatFolder:
	case client.TypeOptionValue:
	case client.TypeChatMember:
	case client.TypeSessions:
	case client.TypeMessage:
	case client.TypeMessages:
	case client.TypeInternalLinkType:
	case client.TypeChatInviteLinkInfo:
	case client.TypeMessageLinkInfo:
	case client.TypeStickers:
	case client.TypeAuthorizationState:

	default:
		log.Printf("WAAAT? update who??? %s, %v", update.GetType(), update)
	}
}
