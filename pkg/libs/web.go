package libs

import (
	"errors"
	"fmt"
	"github.com/alexbilevskiy/tgWatch/pkg/config"
	"github.com/alexbilevskiy/tgWatch/pkg/structs"
	"github.com/zelenin/go-tdlib/client"
	"html/template"
	"log"
	"net/http"
	"strconv"
)

var verbose bool = false
var currentAcc int64

func InitWeb() {
	server := &http.Server{
		Addr:    config.Config.WebListen,
		Handler: HttpHandler{},
	}
	go server.ListenAndServe()
}

func renderTemplates(req *http.Request, w http.ResponseWriter, templateData interface{}, templates ...string) {
	var t *template.Template
	var errParse error
	if verbose {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write(jsonMarshalPretty(templateData))
		if err != nil {
			log.Printf("failed writing debug body: %s", err.Error())
		}
		return
	} else {
		t, errParse = template.New(`base.gohtml`).Funcs(template.FuncMap{
			"formValue": func(key string) string {
				return template.HTMLEscapeString(req.FormValue(key))
			},
			"safeHTML": func(b string) template.HTML {
				return template.HTML(b)
			},
			"renderText": func(text *client.FormattedText) template.HTML {
				return template.HTML(renderText(text))
			},
			"accountsList": func() map[int64]structs.Account {

				return Accounts
			},
			"isMe": func(chatId int64) bool {
				if chatId == me[currentAcc].Id {

					return true
				}

				return false
			},
			"isCurrentAcc": func(acc int64) bool {
				if _, ok := Accounts[currentAcc]; !ok {
					return false
				}
				if acc == Accounts[currentAcc].Id {

					return true
				}

				return false
			},
			"chatInfoLocal": func(chatIdstr string) structs.ChatInfo {
				chatId, _ := strconv.ParseInt(chatIdstr, 10, 64)
				localChat, err := GetChat(currentAcc, chatId, false)
				if err == nil {

					return structs.ChatInfo{ChatId: chatId, ChatName: "_NOT_FOUND_"}
				}

				return buildChatInfoByLocalChat(localChat)
			},
			"chatInfo": func(chatIdstr string) structs.ChatInfo {
				chatId, _ := strconv.ParseInt(chatIdstr, 10, 64)
				c, err := GetChat(currentAcc, chatId, false)
				if err != nil {
					user, err := GetUser(currentAcc, chatId)
					if err != nil {
						return structs.ChatInfo{ChatId: chatId, ChatName: fmt.Sprintf("ERROR: %s", err.Error())}
					}

					return structs.ChatInfo{ChatId: chatId, ChatName: getUserFullname(user)}
				}

				return buildChatInfoByLocalChat(c)
			},
			"GetLink": func(chatId int64, messageId int64) string {
				return GetLink(currentAcc, chatId, messageId)
			},
			"DateTime": func(date int32) string {
				return FormatDateTime(date)
			},
			"Date": func(date int32) string {
				return FormatDate(date)
			},
			"Time": func(date int32) string {
				return FormatTime(date)
			},
			"SetNestedMsg": func(info structs.MessageInfo, text *client.FormattedText, simple string, attachments []structs.MessageAttachment) structs.MessageInfo {
				info.FormattedText = text
				info.SimpleText = simple
				info.Attachments = attachments

				return info
			},
			"dict": func(values ...interface{}) (map[string]interface{}, error) {
				if len(values)%2 != 0 {
					return nil, errors.New("invalid dict call")
				}
				dict := make(map[string]interface{}, len(values)/2)
				for i := 0; i < len(values); i += 2 {
					key, ok := values[i].(string)
					if !ok {
						return nil, errors.New("dict keys must be strings")
					}
					dict[key] = values[i+1]
				}
				return dict, nil
			},
		},
		).ParseFiles(templates...)
	}
	if errParse != nil {
		fmt.Printf("Error tpl: %s\n", errParse)

		return
	}

	var err error
	err = t.Execute(w, templateData)

	if err != nil {
		fmt.Printf("Error tpl: %s\n", err)

		return
	}
}
