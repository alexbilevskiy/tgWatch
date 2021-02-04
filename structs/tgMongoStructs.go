package structs

import (
	"go-tdlib/client"
)

type TgUpdate struct {
	T    string
	Time int32
	Upd  interface{}
	Raw []byte
}

type TgUpdateNewMessage struct {
	T    string
	Time int32
	Upd  client.UpdateNewMessage
	Raw []byte
}

