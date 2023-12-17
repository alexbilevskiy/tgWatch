package structs

type ChatInfo struct {
	ChatId        int64
	ChatName      string
	Username      string
	Type          string
	HasTopics     bool
	CountUnread   int32
	CountMessages int32
}
type Index struct {
	T string
}
type Overview struct {
	T     string
	Chats []ChatInfo
}
type JSON struct {
	JSON string
}

type ChatHistoryOnline struct {
	T              string
	Chat           ChatInfo
	FirstMessageId int64
	LastMessageId  int64
	NextOffset     int64
	PrevOffset     int64
	Messages       []MessageInfo
}

type SingleMessage struct {
	T       string
	Chat    ChatInfo
	Message MessageInfo
}

type ChatFullInfo struct {
	T       string
	Chat    interface{}
	ChatRaw string
}

type Messages struct {
	T           string
	Messages    interface{}
	MessagesRaw string
}

type OptionsList struct {
	T       string
	Options map[string]TdlibOption
}

type SessionsList struct {
	T           string
	Sessions    interface{}
	SessionsRaw string
}

type NewAccountState struct {
	T        string
	Phone    string
	Code     string
	Password string
	State    string
}
