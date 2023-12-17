package libs

import (
	"encoding/base64"
	"github.com/alexbilevskiy/tgWatch/pkg/structs"
	"log"
	"net/http"
	"regexp"
	"strconv"
)

type HttpHandler struct{}

func (h HttpHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	log.Printf("HTTP: %s", req.URL.Path)
	if tryFile(req, res) {
		return
	}

	err := req.ParseForm()
	if err != nil {
		errorResponse(structs.WebError{T: "Unknown error", Error: err.Error()}, 504, req, res)
		return
	}

	verbose = false
	if req.FormValue("a") == "1" {
		verbose = true
	}

	action := regexp.MustCompile(`^/([a-z]*?)(?:$|/.+$)`).FindStringSubmatch(req.URL.Path)
	if action == nil {
		errorResponse(structs.WebError{T: "Not found", Error: req.URL.Path}, 404, req, res)

		return
	}

	if action[1] == "new" {
		processAddAccount(req, res)

		return
	}

	if detectAccount(req, res) == false {

		return
	}

	switch action[1] {
	case "":
		renderTemplates(req, res, nil, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/index.gohtml`)
		return
	case "m":
		r := regexp.MustCompile(`^/m/(-?\d+)/(\d+)$`)
		m := r.FindStringSubmatch(req.URL.Path)
		if m == nil {
			errorResponse(structs.WebError{T: "Not found", Error: req.URL.Path}, 404, req, res)

			return
		}
		chatId, _ := strconv.ParseInt(m[1], 10, 64)
		messageId, _ := strconv.ParseInt(m[2], 10, 64)
		processTgSingleMessage(chatId, messageId, req, res)
		return
	case "j":
		processTgJournal(req, res)
		return
	case "l":
		processTgChatList(req, res)
		return
	case "li":
		processTgLink(req, res)
		return
	case "to":
		processTdlibOptions(req, res)
		return
	case "as":
		processTgActiveSessions(req, res)
		return
	case "c":
		r := regexp.MustCompile(`^/c/(-?\d+)$`)
		m := r.FindStringSubmatch(req.URL.Path)
		if m == nil {
			errorResponse(structs.WebError{T: "Not found", Error: req.URL.Path}, 404, req, res)

			return
		}
		chatId, _ := strconv.ParseInt(m[1], 10, 64)
		processTgChatInfo(chatId, req, res)

		return
	case "h":
		r := regexp.MustCompile(`^/h/?(-?\d+)?($|/)`)
		m := r.FindStringSubmatch(req.URL.Path)
		if m == nil {
			errorResponse(structs.WebError{T: "Not found", Error: req.URL.Path}, 404, req, res)

			return
		}
		chatId, _ := strconv.ParseInt(m[1], 10, 64)
		if m[1] == "" {
			chatId = me[currentAcc].Id
		}

		ids := req.FormValue("ids")
		if ids != "" {
			processTgMessagesByIds(chatId, req, res)
		} else {
			processTgChatHistory(chatId, req, res)
		}

		return
	case "ho":
		r := regexp.MustCompile(`^/ho/?(-?\d+)?($|/)`)
		m := r.FindStringSubmatch(req.URL.Path)
		if m == nil {
			errorResponse(structs.WebError{T: "Not found", Error: req.URL.Path}, 404, req, res)

			return
		}
		chatId, _ := strconv.ParseInt(m[1], 10, 64)
		if m[1] == "" {
			chatId = me[currentAcc].Id
		}
		processTgChatHistoryOnline(chatId, req, res)

		return
	case "f":
		r := regexp.MustCompile(`^/f/([\w\-_]+)$`)
		m := r.FindStringSubmatch(req.URL.Path)
		if m == nil || m[1] == "" {
			errorResponse(structs.WebError{T: "Not found", Error: req.URL.Path}, 404, req, res)

			return
		}

		file, err := DownloadFileByRemoteId(currentAcc, m[1])

		if err != nil {
			errorResponse(structs.WebError{T: "Attachment error", Error: err.Error()}, 502, req, res)

			return
		}
		if verbose {
			renderTemplates(req, res, file)

			return
		}
		if file.Local.Path != "" {
			res.Header().Add("X-Local-path", base64.StdEncoding.EncodeToString([]byte(file.Local.Path)))
			http.ServeFile(res, req, file.Local.Path)

			return
		}

		errorResponse(structs.WebError{T: "Invalid file", Error: file.Extra}, 504, req, res)

		return

	case "s":
		processSettings(req, res)
		return
	case "delete":
		r := regexp.MustCompile(`^/delete/(-?\d+)$`)
		m := r.FindStringSubmatch(req.URL.Path)
		if m == nil {
			errorResponse(structs.WebError{T: "Not found", Error: req.URL.Path}, 404, req, res)

			return
		}

		chatId, _ := strconv.ParseInt(m[1], 10, 64)

		processTgDelete(chatId, req, res)

		return
	default:
		errorResponse(structs.WebError{T: "Not found", Error: req.URL.Path}, 404, req, res)

		return
	}
}

func detectAccount(req *http.Request, res http.ResponseWriter) bool {
	accCookie, err := req.Cookie("acc")
	if err != nil {
		log.Printf("Cookie errror: %s", err.Error())

		currentAcc = -1
		renderTemplates(req, res, nil, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/account_select.gohtml`)

		return false
	}
	currentAcc, err = strconv.ParseInt(accCookie.Value, 10, 64)
	if err != nil {
		errorResponse(structs.WebError{T: "Invalid account", Error: err.Error()}, 504, req, res)

		return false
	}

	if _, ok := Accounts[currentAcc]; !ok {
		errorResponse(structs.WebError{T: "Invalid account", Error: "no such account"}, 504, req, res)

		return false
	}

	cookie := http.Cookie{Name: "acc", Value: strconv.FormatInt(currentAcc, 10), Path: "/"}
	http.SetCookie(res, &cookie)

	return true
}
