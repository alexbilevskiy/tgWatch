package account

import (
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
	ListenUpdates()
	Close()

	GetChat(chatId int64, force bool) (*client.Chat, error)
	GetUser(userId int64) (*client.User, error)
	GetSuperGroup(sgId int64) (*client.Supergroup, error)
	GetBasicGroup(groupId int64) (*client.BasicGroup, error)
	GetGroupsInCommon(userId int64) (*client.Chats, error)
	DownloadFile(id int32) (*client.File, error)
	DownloadFileByRemoteId(id string) (*client.File, error)
	GetLink(chatId int64, messageId int64) string
	AddChatsToFolder(chats []int64, folder int32) error
	SendMessage(text string, chatId int64, replyToMessageId *int64)
	GetLinkInfo(link string) (client.InternalLinkType, interface{}, error)
	GetMessage(chatId int64, messageId int64) (*client.Message, error)
	LoadChatHistory(chatId int64, fromMessageId int64, offset int32) (*client.Messages, error)
	MarkJoinAsRead(chatId int64, messageId int64)
	GetTdlibOption(optionName string) (client.OptionValue, error)
	GetActiveSessions() (*client.Sessions, error)
	GetChatHistory(chatId int64, lastId int64) (*client.Messages, error)
	DeleteMessages(chatId int64, messageIds []int64) (*client.Ok, error)
	GetChatMember(chatId int64) (*client.ChatMember, error)
	GetScheduledMessages(chatId int64) (*client.Messages, error)
	ScheduleForwardedMessage(targetChatId int64, fromChatId int64, messageIds []int64, sendAtDate int32, sendCopy bool) (*client.Messages, error)
	GetCustomEmoji(customEmojisIds []int64) (*client.Stickers, error)

	GetSenderName(sender client.MessageSender) string
	GetSenderObj(sender client.MessageSender) (interface{}, error)
	GetChatName(chatId int64) string
	GetChatUsername(chatId int64) string

	SaveChatFilters(chatFoldersUpdate *client.UpdateChatFolders)
	SaveChatAddedToList(upd *client.UpdateChatAddedToList)
	RemoveChatRemovedFromList(upd *client.UpdateChatRemovedFromList)
	LoadChatsList(listId int32)
	GetChatFolders() []db.ChatFilter
	GetLocalChats() map[int64]*client.Chat

	GetStorage() tdlib.TdStorageInterface
}

func NewAccount(cfg *config.Config, tdMongo *db.TdMongo, dbData *db.DbAccountData) *Account {
	tdApi := tdlib.NewTdApi(cfg, dbData, tdMongo)
	me := tdApi.RunTdlib()

	acc := &Account{
		Username: tdlib.GetUsername(me.Usernames),
		DbData:   dbData,
		Me:       me,
		TdApi:    tdApi,
	}

	return acc
}

func (acc *Account) Run() {
	go acc.TdApi.ListenUpdates()
}
