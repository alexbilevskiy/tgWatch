# TgWatch

Initially this client was intended for logging and viewing deleted and updated messages in telegram chats.   
Now it's more like a telegram explorer - view raw messages and chats, files, etc.

Features:
* chats list (folders included) with message counters 
* chat history (only *local* history)
* configurable list of ignored chats and senders
* multi-account support (beta!)
* files downloader

Next ideas:
* mute chats by conditions (how to define rules?)
* merge multiple outgoing one-line messages into one
* auto-respond
* decode voice messages in private chats (send recognized text as response)
* load remote chat history
* replies

BUGS:
* uses custom go-tdlib fork to support multiple clients
* new account log-in process is kinda tricky and requires restart after successful login

### install:
`go build`
### dependencies:
* [tdlib](https://tdlib.github.io/td/build.html?language=Go)
* [golang mongo driver](https://pkg.go.dev/go.mongodb.org/mongo-driver)
* [golang tdlib wrapper](https://github.com/zelenin/go-tdlib)
