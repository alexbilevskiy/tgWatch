# TgWatch

Initially this client was intended for logging and viewing deleted and updated messages in telegram chats.   
Now it's more like a telegram explorer - view raw messages and chats, files downloader, etc.
Includes chat history viewer (can't reply though)

Next ideas: 
* mute chats by conditions. For example, mute chat for 6 hours if someone sends birthday congratulations card
* merge multiple outgoing one-line messages into one
* auto-respond
* decode voice messages in private chats (send recognized text as response)
* multi-account support


### install:
`go build`
### dependencies:
* [tdlib](https://tdlib.github.io/td/build.html?language=Go)
* [golang mongo driver](https://pkg.go.dev/go.mongodb.org/mongo-driver)
* [golang tdlib wrapper](https://github.com/zelenin/go-tdlib)
