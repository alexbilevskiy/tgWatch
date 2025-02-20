package web

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/alexbilevskiy/tgWatch/internal/account"
	"github.com/alexbilevskiy/tgWatch/internal/helpers"
	"github.com/alexbilevskiy/tgWatch/internal/tdlib"
	"github.com/zelenin/go-tdlib/client"
)

func renderTemplates(req *http.Request, w http.ResponseWriter, templateData interface{}, templates ...string) {
	var t *template.Template
	var errParse error
	if req.Context().Value("verbose").(bool) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write(helpers.JsonMarshalPretty(templateData))
		if err != nil {
			log.Printf("failed writing debug body: %s", err.Error())
		}
		return
	} else {
		t, errParse = template.New(`base.gohtml`).Funcs(funcMap(req)).ParseFiles(templates...)
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

func funcMap(req *http.Request) template.FuncMap {
	currentAcc := req.Context().Value("current_acc").(int64)
	return template.FuncMap{
		"formValue": func(key string) string {
			return template.HTMLEscapeString(req.FormValue(key))
		},
		"safeHTML": func(b string) template.HTML {
			return template.HTML(b)
		},
		"accountsList": func() []*account.Account {
			accounts := make([]*account.Account, 0)
			account.AS.Range(func(key any, value any) bool {
				accounts = append(accounts, value.(*account.Account))
				return true
			})
			return accounts
		},
		"isMe": func(chatId int64) bool {
			if chatId == account.AS.Get(currentAcc).DbData.Id {

				return true
			}

			return false
		},
		"isCurrentAcc": func(acc int64) bool {
			if acc == account.AS.Get(currentAcc).DbData.Id {

				return true
			}

			return false
		},
		"renderText": func(text *client.FormattedText) template.HTML {
			return template.HTML(RenderText(text))
		},
		"chatInfoLocal": func(chatIdstr string) ChatInfo {
			chatId, _ := strconv.ParseInt(chatIdstr, 10, 64)
			localChat, err := account.AS.Get(currentAcc).TdApi.GetChat(chatId, false)
			if err == nil {

				return ChatInfo{ChatId: chatId, ChatName: "_NOT_FOUND_"}
			}

			return buildChatInfoByLocalChat(req.Context(), localChat)
		},
		"chatInfo": func(chatIdstr string) ChatInfo {
			chatId, _ := strconv.ParseInt(chatIdstr, 10, 64)
			c, err := account.AS.Get(currentAcc).TdApi.GetChat(chatId, false)
			if err != nil {
				user, err := account.AS.Get(currentAcc).TdApi.GetUser(chatId)
				if err != nil {
					return ChatInfo{ChatId: chatId, ChatName: fmt.Sprintf("ERROR: %s", err.Error())}
				}

				return ChatInfo{ChatId: chatId, ChatName: tdlib.GetUserFullname(user)}
			}

			return buildChatInfoByLocalChat(req.Context(), c)
		},
		"GetLink": func(chatId int64, messageId int64) string {
			return account.AS.Get(currentAcc).TdApi.GetLink(chatId, messageId)
		},
		"DateTime": func(date int32) string {
			return helpers.FormatDateTime(date)
		},
		"Date": func(date int32) string {
			return helpers.FormatDate(date)
		},
		"Time": func(date int32) string {
			return helpers.FormatTime(date)
		},
		"SetNestedMsg": func(info MessageInfo, text *client.FormattedText, simple string, attachments []MessageAttachment) MessageInfo {
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
	}
}
