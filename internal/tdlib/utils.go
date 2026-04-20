package tdlib

import (
	"fmt"

	"github.com/alexbilevskiy/tgwatch/internal/config"
	"github.com/zelenin/go-tdlib/client"
)

var TdlibOptions map[string]TdlibOption

func GetChatIdBySender(sender client.MessageSender) int64 {
	senderChatId := int64(0)
	if sender.MessageSenderConstructor() == client.ConstructorMessageSenderChat {
		senderChatId = sender.(*client.MessageSenderChat).ChatId
	} else if sender.MessageSenderConstructor() == client.ConstructorMessageSenderUser {
		senderChatId = int64(sender.(*client.MessageSenderUser).UserId)
	}

	return senderChatId
}

func GetUserFullname(user *client.User) string {
	name := ""
	if user.FirstName != "" {
		name = user.FirstName
	}
	if user.LastName != "" {
		name = fmt.Sprintf("%s %s", name, user.LastName)
	}
	un := GetUsername(user.Usernames)
	if un != "" {
		name = fmt.Sprintf("%s (@%s)", name, un)
	}
	if name == "" {
		name = fmt.Sprintf("no_name %d", user.Id)
	}
	return name
}

func GetUsername(usernames *client.Usernames) string {
	if usernames == nil {
		return ""
	}
	if len(usernames.ActiveUsernames) == 0 {
		return ""
	}
	if len(usernames.ActiveUsernames) > 1 {
		//log.Printf("whoa, multiple usernames? %s", helpers.JsonMarshalStr(usernames.ActiveUsernames))
		return usernames.ActiveUsernames[0]
	}

	return usernames.ActiveUsernames[0]
}

func LoadOptionsList() error {
	var opts map[string]TdlibOption
	opts = make(map[string]TdlibOption)
	err := config.UnmarshalJsonFile("tdlib_options.json", &opts)
	if err != nil {
		return fmt.Errorf("failed to read tdlib options desc: %w", err)
	}
	TdlibOptions = opts

	return nil
}

func buildProxyOption(cfg *config.Config) client.Option {
	if cfg.ProxyUser != "" {
		proxy := client.Proxy{Type: &client.ProxyTypeSocks5{Username: cfg.ProxyUser, Password: cfg.ProxyPass}, Server: cfg.ProxyHost, Port: cfg.ProxyPort}
		proxyReq := client.AddProxyRequest{Proxy: &proxy, Enable: true}
		return client.WithProxy(&proxyReq)
	} else if cfg.ProxySecret != "" {
		proxy := client.Proxy{Type: &client.ProxyTypeMtproto{Secret: cfg.ProxySecret}, Server: cfg.ProxyHost, Port: cfg.ProxyPort}
		proxyReq := client.AddProxyRequest{Proxy: &proxy, Enable: true}
		return client.WithProxy(&proxyReq)
	} else {
		return nil
	}

}