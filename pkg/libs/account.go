package libs

import (
	"github.com/alexbilevskiy/tgWatch/pkg/consts"
	"github.com/alexbilevskiy/tgWatch/pkg/libs/mongo"
	"github.com/alexbilevskiy/tgWatch/pkg/libs/tdlib"
	"github.com/alexbilevskiy/tgWatch/pkg/libs/tdlib/tdAccount"
	"github.com/zelenin/go-tdlib/client"
	"log"
	"sync"
	"time"
)

type Account struct {
	DbData   *mongo.DbAccountData
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
	GetCustomEmoji(customEmojisIds []client.JsonInt64) (*client.Stickers, error)

	GetSenderName(sender client.MessageSender) string
	GetSenderObj(sender client.MessageSender) (interface{}, error)
	GetChatName(chatId int64) string
	GetChatUsername(chatId int64) string

	SaveChatFilters(chatFoldersUpdate *client.UpdateChatFolders)
	SaveChatAddedToList(upd *client.UpdateChatAddedToList)
	RemoveChatRemovedFromList(upd *client.UpdateChatRemovedFromList)
	LoadChatsList(listId int32)
	GetChatFolders() []mongo.ChatFilter
	GetLocalChats() map[int64]*client.Chat

	GetStorage() tdlib.TdStorageInterface
}

func (acc *Account) RunAccount() {
	tdMongo := mongo.TdMongo{}
	tdMongo.Init(acc.DbData.DbPrefix, acc.DbData.Phone)

	tdlibClient, me := tdAccount.RunTdlib(acc.DbData)
	acc.Username = tdlib.GetUsername(me.Usernames)
	acc.Me = me

	tdApi := tdlib.TdApi{}
	tdApi.Init(acc.DbData, tdlibClient, &tdMongo)

	acc.TdApi = &tdApi

	go tdApi.ListenUpdates()
}

var AS AccountStorage

type AccountStorage struct {
	accounts sync.Map
}

func (as *AccountStorage) Create(mongoAcc *mongo.DbAccountData) {
	acc := &Account{
		DbData: mongoAcc,
	}
	as.accounts.Store(acc.DbData.Id, acc)
}

func (as *AccountStorage) Get(accId int64) *Account {
	acc, ok := as.accounts.Load(accId)
	if !ok {
		return nil
	}

	return acc.(*Account)
}

func (as *AccountStorage) Delete(accId int64) {

	as.accounts.Delete(accId)
}

func (as *AccountStorage) Range(f func(key any, value any) bool) {

	as.accounts.Range(f)
}

func (as *AccountStorage) RunOne(phone string) {
	accounts := mongo.LoadAccounts(phone)
	for _, mongoAcc := range accounts {
		if mongoAcc.Status != consts.AccStatusActive {
			log.Printf("wont run account %d, because its not active yet: `%s`", mongoAcc.Id, mongoAcc.Status)
			continue
		}
		log.Printf("create account %d", mongoAcc.Id)
		AS.Create(mongoAcc)
		AS.Get(mongoAcc.Id).RunAccount()
	}
}

func (as *AccountStorage) Run() {
	for {
		accounts := mongo.LoadAccounts("")
		for _, mongoAcc := range accounts {
			if AS.Get(mongoAcc.Id) != nil {
				if mongoAcc.Status != consts.AccStatusActive {
					//not implemented actually. No one updates status to non-active axcept when upating manually in DB
					log.Printf("need to stop account %d, because it became active: `%s`", mongoAcc.Id, mongoAcc.Status)
					AS.Get(mongoAcc.Id).TdApi.Close()
					AS.Delete(mongoAcc.Id)
				} else {
					//already running
				}
				continue
			}
			if mongoAcc.Status != consts.AccStatusActive {
				log.Printf("wont run account %d, because its not active yet: `%s`", mongoAcc.Id, mongoAcc.Status)
				continue
			}
			log.Printf("create account %d", mongoAcc.Id)
			AS.Create(mongoAcc)
			AS.Get(mongoAcc.Id).RunAccount()
		}
		time.Sleep(5 * time.Second)
	}
}
