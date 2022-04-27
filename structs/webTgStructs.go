package structs

import "go-tdlib/client"

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
	Deleted       bool
	DeletedAt     int32
	Edited        bool
	ContentRaw    interface{}
}

type MessageEditedInfo struct {
	T             string
	MessageId     int64
	FormattedText *client.FormattedText
	SimpleText    string
	Attachments   []MessageAttachment
	Date          int32
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
