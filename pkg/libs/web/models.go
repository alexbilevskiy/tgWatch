package web

import (
	"github.com/alexbilevskiy/tgWatch/pkg/libs/tdlib"
	"github.com/zelenin/go-tdlib/client"
)

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
	Chat    ChatInfo
	ChatRaw string
}

type Messages struct {
	T           string
	Messages    interface{}
	MessagesRaw string
}

type OptionsList struct {
	T       string
	Options map[string]tdlib.TdlibOption
}

type SessionsList struct {
	T           string
	Sessions    interface{}
	SessionsRaw string
}

type MessageInfo struct {
	T             string
	MessageId     int64
	Date          int32
	DateTimeStr   string
	DateStr       string
	TimeStr       string
	ChatId        int64
	ChatName      string
	SenderId      int64
	SenderName    string
	MediaAlbumId  int64
	FormattedText *client.FormattedText
	SimpleText    string
	Attachments   []MessageAttachment
	Edited        bool
	ContentRaw    interface{}
}

type MessageError struct {
	T         string
	MessageId int64
	Error     string
}

type MessageAttachment struct {
	T         string
	Id        string
	Link      []string
	Thumb     string
	ThumbLink string
	Name      string
}

type MessageAttachmentError struct {
	T     string
	Id    string
	Error string
}

type ChatFolder struct {
	T     string
	Id    int32
	Title string
}

type ChatList struct {
	T              string
	SelectedFolder int32
	PartnerChat    ChatInfo
	ChatFolders    []ChatFolder
	Chats          []ChatInfo
}
type MessageTextContent struct {
	FormattedText *client.FormattedText
	Text          string
}

type WebError struct {
	T     string
	Error string
}
