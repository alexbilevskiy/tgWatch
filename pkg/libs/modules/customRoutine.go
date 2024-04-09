package modules

import "github.com/zelenin/go-tdlib/client"

// @TODO: create some kind of lua integration to allow writing custom message processing plugins without need to recompile
func CustomNewMessageRoutine(acc int64, tdlibClient *client.Client, update *client.UpdateNewMessage) {
}

func CustomMessageContentRoutine(acc int64, tdlibClient *client.Client, update *client.UpdateMessageContent) {
}
