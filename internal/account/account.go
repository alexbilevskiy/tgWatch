package account

import (
	"context"
	"fmt"

	"github.com/alexbilevskiy/tgWatch/internal/config"
	"github.com/alexbilevskiy/tgWatch/internal/db"
	"github.com/alexbilevskiy/tgWatch/internal/tdlib"
	"github.com/zelenin/go-tdlib/client"
)

type Account struct {
	DbData   *db.DbAccountData
	Username string
	TdApi    tdApiInterface
	Me       *client.User
}

type tdApiInterface interface {
	RunTdlib(ctx context.Context) (*client.User, error)
	Close(ctx context.Context)

	GetChat(ctx context.Context, chatId int64, force bool) (*client.Chat, error)
	GetUser(ctx context.Context, userId int64) (*client.User, error)
	GetSuperGroup(ctx context.Context, sgId int64) (*client.Supergroup, error)
	GetBasicGroup(ctx context.Context, groupId int64) (*client.BasicGroup, error)
	GetGroupsInCommon(ctx context.Context, userId int64) (*client.Chats, error)
	DownloadFile(ctx context.Context, id int32) (*client.File, error)
	DownloadFileByRemoteId(ctx context.Context, id string) (*client.File, error)
	GetLink(ctx context.Context, chatId int64, messageId int64) string
	AddChatsToFolder(ctx context.Context, chats []int64, folder int32) error
	SendMessage(ctx context.Context, text string, chatId int64, replyToMessageId *int64)
	GetLinkInfo(ctx context.Context, link string) (client.InternalLinkType, interface{}, error)
	GetMessage(ctx context.Context, chatId int64, messageId int64) (*client.Message, error)
	LoadChatHistory(ctx context.Context, chatId int64, fromMessageId int64, offset int32) (*client.Messages, error)
	MarkJoinAsRead(ctx context.Context, chatId int64, messageId int64)
	GetTdlibOption(optionName string) (client.OptionValue, error)
	GetActiveSessions(ctx context.Context) (*client.Sessions, error)
	GetChatHistory(ctx context.Context, chatId int64, lastId int64) (*client.Messages, error)
	DeleteMessages(ctx context.Context, chatId int64, messageIds []int64) (*client.Ok, error)
	GetChatMember(ctx context.Context, chatId int64) (*client.ChatMember, error)
	GetScheduledMessages(ctx context.Context, chatId int64) (*client.Messages, error)
	ScheduleForwardedMessage(ctx context.Context, targetChatId int64, fromChatId int64, messageIds []int64, sendAtDate int32, sendCopy bool) (*client.Messages, error)
	GetCustomEmoji(ctx context.Context, customEmojisIds []int64) (*client.Stickers, error)

	GetSenderName(ctx context.Context, sender client.MessageSender) string
	GetSenderObj(ctx context.Context, sender client.MessageSender) (interface{}, error)
	GetChatName(ctx context.Context, chatId int64) string
	GetChatUsername(ctx context.Context, chatId int64) string

	SaveChatFilters(ctx context.Context, chatFoldersUpdate *client.UpdateChatFolders)
	SaveChatAddedToList(ctx context.Context, upd *client.UpdateChatAddedToList)
	RemoveChatRemovedFromList(ctx context.Context, upd *client.UpdateChatRemovedFromList)
	LoadChatsList(ctx context.Context, listId int32)
	GetChatFolders() []db.ChatFilter
	GetLocalChats() map[int64]*client.Chat

	GetStorage() tdlib.TdStorageInterface
}

func NewAccount(cfg *config.Config, tdMongo *db.TdMongo, dbData *db.DbAccountData) *Account {
	tdApi := tdlib.NewTdApi(cfg, dbData, tdMongo)

	acc := &Account{
		DbData: dbData,
		TdApi:  tdApi,
	}

	return acc
}

func (a *Account) Run(ctx context.Context) error {
	me, err := a.TdApi.RunTdlib(ctx)
	if err != nil {
		return fmt.Errorf("run tdlib (%s): %w", a.DbData.Phone, err)
	}
	username := tdlib.GetUsername(me.Usernames)
	a.Me = me
	a.Username = username

	return nil
}
