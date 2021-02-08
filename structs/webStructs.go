package structs

type MessageInfo struct {
	T          string
	MessageId  int64
	Date       int32
	DateStr    string
	ChatId     int64
	ChatName   string
	SenderId   int64
	SenderName string
	Content    string
	ContentRaw interface{}
}

type MessageNewContent struct {
	T          string
	MessageId  int64
	Content    string
	ContentRaw interface{}
}

type MessageEditedMeta struct {
	T         string
	MessageId int64
	Date      int32
	DateStr   string
}

type MessageError struct {
	T         string
	MessageId int64
	Error	  string
}
