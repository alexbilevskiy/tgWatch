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
type JSON struct {
	JSON string
}
