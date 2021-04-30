package structs

type ChatInfo struct {
	ChatId        int64
	ChatName      string
	CountTotal    int32
	CountMessages int32
	CountEdits    int32
	CountDeletes  int32
}
type JournalItem struct {
	T         string
	Time      int32
	Date      string
	MessageId []int64
	Chat      ChatInfo
	From      ChatInfo
	Link      string
	IntLink   string
	Message   string
	Error     string
}
type Journal struct {
	T string
	J []JournalItem
}
type Index struct {
	T string
}
type Overview struct {
	T string
	Chats []ChatInfo
}
type JSON struct {
	JSON string
}

type ChatHistory struct {
	T          string
	Chat       ChatInfo
	Limit      int64
	Offset     int64
	NextOffset int64
	PrevOffset int64
	Messages   []MessageInfo
}

type ChatFullInfo struct {
	T       string
	Chat    interface{}
	ChatRaw string
}

type DeletedMessages struct {
	T          string
	Messages    interface{}
	MessagesRaw string
}

type OptionsList struct {
	T       string
	Options map[string]TdlibOption
}

