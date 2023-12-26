# TgWatch

Initially this client was intended for logging and viewing deleted and updated messages in telegram chats, but this functionality is against telegram ToS, so it was removed.
Now it's more like a telegram explorer - view raw messages and chats, files, responses, etc.

Implemented telegram features:
* chats list (folders included) with message counters 
* chat history (only *online* history - no local copy)
* sending messages
* profile info
* active sessions
* multi-account support

Next ideas:
* mute chats by conditions (how to define rules? "all chats in folder "work" except these two")
* merge multiple outgoing one-line messages into one
* auto-respond 

BUGS:
* uses custom go-tdlib fork
* new account log-in process is kinda tricky and requires restart after successful login

### install:
* install [tdlib](https://tdlib.github.io/td/build.html?language=Go)
* use forked go-tdlib `go mod edit -replace="github.com/zelenin/go-tdlib=github.com/alexbilevskiy/go-tdlib@master"`
* compile with `-stdlib=libstdc++`
* `run.sh`
* Dockerfile also provided

### dependencies:
* [tdlib](https://tdlib.github.io/td/build.html?language=Go)
* [golang mongo driver](https://pkg.go.dev/go.mongodb.org/mongo-driver)
* [golang tdlib wrapper](https://github.com/zelenin/go-tdlib)
