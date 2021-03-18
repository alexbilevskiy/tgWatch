package structs

type ChatInfo struct {
	ChatId   int64
	ChatName string
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
type OverviewItem struct {
	Chat          ChatInfo
	CountTotal    int32
	CountMessages int32
	CountEdits    int32
	CountDeletes  int32
}
type Overview struct {
	T string
	O []OverviewItem
}
type JSON struct {
	JSON string
}

type ChatHistory struct {
	T        string
	Chat     ChatInfo
	Messages []MessageInfo
}

type ChatFullInfo struct {
	T       string
	Chat    interface{}
	ChatRaw string
}