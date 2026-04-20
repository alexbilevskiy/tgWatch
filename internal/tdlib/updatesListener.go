package tdlib

import (
	"context"

	"github.com/alexbilevskiy/tgwatch/internal/modules"
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
		case client.ConstructorUpdateEmojiChatThemes:
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
		case client.ConstructorUpdateVideoPublished:
		case client.ConstructorUpdateChatEmojiStatus:
		case client.ConstructorUpdateTrustedMiniAppBots:
		case client.ConstructorUpdateGroupCallMessageLevels:
		case client.ConstructorUpdateOwnedTonCount:

		case client.ConstructorUpdateMessageInteractionInfo:
		case client.ConstructorUpdateChatTitle:
		case client.ConstructorUpdateNewChat:
		case client.ConstructorUpdateConnectionState:
		case client.ConstructorUpdateChatAction:
		case client.ConstructorUpdateFile:
			upd := update.(*client.UpdateFile)
			if upd.File.Local.IsDownloadingActive {
				t.log.Info("file downloading", "bytes", upd.File.Local.DownloadedSize, "total", upd.File.ExpectedSize)
			} else {
				t.log.Info("file downloaded", "bytes", upd.File.Local.DownloadedSize, "path", upd.File.Local.Path)
			}
		case client.ConstructorUpdateChatAvailableReactions:

		case client.ConstructorUpdateMessageSendSucceeded:
			upd := update.(*client.UpdateMessageSendSucceeded)
			t.sentMessages.Store(upd.OldMessageId, upd.Message)
		case client.ConstructorUpdateMessageSendFailed:
			upd := update.(*client.UpdateMessageSendFailed)
			//@TODO: also put in t.sentMessages
			t.log.Error("failed to send message", "virtual_id", upd.OldMessageId, "error", upd.Error)

		case client.ConstructorUpdateChatHasProtectedContent:
			upd := update.(*client.UpdateChatHasProtectedContent)
			t.log.Info("Chat now has protected content", "chat_id", upd.ChatId, "name", t.GetChatName(ctx, upd.ChatId), "value", upd.HasProtectedContent)

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
				go t.MarkJoinAsRead(ctx, upd.Message.ChatId, upd.Message.Id)
			}

			go modules.CustomNewMessageRoutine(ctx, t.dbData.Id, t.tdlibClient, upd)

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

			go modules.CustomMessageContentRoutine(ctx, t.dbData.Id, t.tdlibClient, upd)

		case client.ConstructorUpdateChatMessageAutoDeleteTime:
			upd := update.(*client.UpdateChatMessageAutoDeleteTime)
			chatName := t.GetChatName(ctx, upd.ChatId)
			t.log.Info("message auto-delete time updated", "chat_id", upd.ChatId, "name", chatName, "value", upd.MessageAutoDeleteTime)

		default:
			t.log.Info("unknown update", "type", typ, "value", update)
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
	case client.TypeFoundPublicPosts:

	default:
		t.log.Info("WAAAT? update who???", "type", update.GetType(), "value", update)
	}
}
