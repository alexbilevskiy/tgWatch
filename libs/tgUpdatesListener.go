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
		if update.GetClass() == client.ClassUpdate {
			t := update.GetType()
			switch t {
			case "updateChatActionBar":
			case "updateChatIsBlocked":
			case "updateChatDraftMessage":
			case "updateUserStatus":
			case "updateChatReadInbox":
			case "updateChatReadOutbox":
			case "updateUnreadMessageCount":
			case "updateUnreadChatCount":
			case "updateMessageInteractionInfo":
			case "updateChatReplyMarkup":
			case "updateChatPermissions":
			case "updateChatNotificationSettings":
			case "updateChatUnreadMentionCount":
			case "updateMessageMentionRead":
			case "updateMessageIsPinned":
			case "updateChatHasScheduledMessages":
			case "updateHavePendingNotifications":
			case "updateRecentStickers":
			case "updateCall":
			case "updateMessageContentOpened":

			case "updateSupergroup":
			case "updateSupergroupFullInfo":
			case "updateBasicGroup":
			case "updateBasicGroupFullInfo":
			case "updateUser":
			case "updateUserFullInfo":
			case "updateChatPhoto":
				break

			case "updateChatTitle":
				upd := update.(*client.UpdateChatTitle)
				log.Printf("Renamed chat id:%d to `%s`", upd.ChatId, upd.Title)

				break
			case "updateNewChat":
				upd := update.(*client.UpdateNewChat)
				localChats[upd.Chat.Id] = upd.Chat
				DLog(fmt.Sprintf("New chat added: %d / %s", upd.Chat.Id, upd.Chat.Title))
				saveAllChatPositions(upd.Chat.Id, upd.Chat.Positions)

				break
			case "updateConnectionState":
				upd := update.(*client.UpdateConnectionState)
				log.Printf("Connection state changed: %s", upd.State.ConnectionStateType())

				break
			case "updateUserChatAction":
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
				log.Printf("User action in chat `%s`: id: %d, name: `%s`, action: %s", GetChatName(upd.ChatId), upd.UserId, userName, upd.Action.ChatActionType())

				break
			case "updateChatLastMessage":
				upd := update.(*client.UpdateChatLastMessage)
				if len(upd.Positions) == 0 {
					break
				}
				saveAllChatPositions(upd.ChatId, upd.Positions)

				break
			case "updateOption":
				upd := update.(*client.UpdateOption)
				log.Printf("Update option %s: %s", upd.Name, JsonMarshalStr(upd.Value))

				break
			case "updateChatPosition":
				upd := update.(*client.UpdateChatPosition)
				saveChatPosition(upd.ChatId, upd.Position)

				break
			case "updateChatFilters":
				upd := update.(*client.UpdateChatFilters)
				SaveChatFilters(upd)

				break

			case "updateDeleteMessages":
				upd := update.(*client.UpdateDeleteMessages)
				if !upd.IsPermanent || upd.FromCache {

					break
				}
				if checkSkippedChat(strconv.FormatInt(upd.ChatId, 10)) || checkChatFilter(upd.ChatId) {

					break
				}

				skipUpdate := 0
				for _, messageId := range upd.MessageIds {
					savedMessage, err := FindUpdateNewMessage(upd.ChatId, messageId)
					if err != nil {

						continue
					}
					if checkSkippedChat(strconv.FormatInt(GetChatIdBySender(savedMessage.Message.Sender), 10)) {
						DLog(fmt.Sprintf("Skip deleted message %d from sender %d, `%s`", messageId, GetChatIdBySender(savedMessage.Message.Sender), GetSenderName(savedMessage.Message.Sender)))
						skipUpdate++

						continue
					}
					if savedMessage.Message.Content == nil {
						log.Printf("Skip deleted message %d with unknown content from %s", messageId, GetChatIdBySender(savedMessage.Message.Sender))

						continue
					}
					if savedMessage.Message.Content.MessageContentType() == "messageChatAddMembers" {
						DLog(fmt.Sprintf("Skip deleted message %d (chat join of user %d)", messageId, GetChatIdBySender(savedMessage.Message.Sender)))
						skipUpdate++

						continue
					}
				}
				if skipUpdate == len(upd.MessageIds) {

					break
				}
				mongoId := SaveUpdate(t, upd, 0)

				chatName := GetChatName(upd.ChatId)
				intLink := fmt.Sprintf("http://%s/d/%d/%s", config.Config.WebListen, upd.ChatId, ImplodeInt(upd.MessageIds))
				count := len(upd.MessageIds)
				DLog(fmt.Sprintf("[%s] DELETED %d Messages from chat: %d, `%s`, %s", mongoId, count, upd.ChatId, chatName, intLink))

				break

			case "updateNewMessage":
				upd := update.(*client.UpdateNewMessage)
				if checkSkippedChat(strconv.FormatInt(upd.Message.ChatId, 10)) || checkChatFilter(upd.Message.ChatId) {

					break
				}
				//senderChatId := GetChatIdBySender(upd.Message.Sender)
				SaveUpdate(t, upd, upd.Message.Date)
				//mongoId := SaveUpdate(t, upd, upd.Message.Date)
				//link := GetLink(tdlibClient, upd.Message.ChatId, upd.Message.Id)
				//chatName := GetChatName(upd.Message.ChatId)
				//intLink := fmt.Sprintf("http://%s/e/%d/%d", config.Config.WebListen, upd.Message.ChatId, upd.Message.Id)
				//log.Printf("[%s] New Message from chat: %d, `%s`, %s, %s", mongoId, upd.Message.ChatId, chatName, link, intLink)

				break
			case "updateMessageEdited":
				upd := update.(*client.UpdateMessageEdited)
				if checkSkippedChat(strconv.FormatInt(upd.ChatId, 10)) || checkChatFilter(upd.ChatId) {

					break
				}

				if upd.ReplyMarkup != nil {
					//messages with buttons - reactions, likes etc

					break
				}
				SaveUpdate(t, upd, upd.EditDate)
				//mongoId := SaveUpdate(t, upd, upd.EditDate)
				//link := GetLink(tdlibClient, upd.ChatId, upd.MessageId)
				//chatName := GetChatName(upd.ChatId)
				//intLink := fmt.Sprintf("http://%s/e/%d/%d", config.Config.WebListen, upd.ChatId, upd.MessageId)
				//log.Printf("[%s] EDITED msg! Chat: %d, msg %d, `%s`, %s, %s", mongoId, upd.ChatId, upd.MessageId, chatName, link, intLink)

				break
			case "updateMessageContent":
				upd := update.(*client.UpdateMessageContent)
				if checkSkippedChat(strconv.FormatInt(upd.ChatId, 10)) || checkChatFilter(upd.ChatId) {

					break
				}
				if upd.NewContent.MessageContentType() == "messagePoll" {
					//dont save "poll" updates - that's just counters, users cannot update polls manually
					break
				}
				mongoId := SaveUpdate(t, upd, 0)

				link := GetLink(upd.ChatId, upd.MessageId)
				chatName := GetChatName(upd.ChatId)
				intLink := fmt.Sprintf("http://%s/e/%d/%d", config.Config.WebListen, upd.ChatId, upd.MessageId)
				DLog(fmt.Sprintf("[%s] EDITED content! Chat: %d, msg %d, %s, %s, %s", mongoId, upd.ChatId, upd.MessageId, chatName, link, intLink))

				break
			case "updateFile":
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
		}
	}
}
