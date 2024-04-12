package web

import (
	"encoding/base64"
	"github.com/alexbilevskiy/tgWatch/pkg/libs"
	"github.com/alexbilevskiy/tgWatch/pkg/structs"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type HttpHandler struct {
	Controller webController
}

func (h HttpHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	log.Printf("HTTP: %s", req.URL.Path)
	if tryFile(req, res) {
		return
	}

	err := req.ParseForm()
	if err != nil {
		h.Controller.errorResponse(structs.WebError{T: "Unknown error", Error: err.Error()}, 504, req, res)
		return
	}

	verbose = false
	if req.FormValue("a") == "1" {
		verbose = true
	}

	action := regexp.MustCompile(`^/([a-z]*?)(?:$|/.+$)`).FindStringSubmatch(req.URL.Path)
	if action == nil {
		h.Controller.errorResponse(structs.WebError{T: "Not found", Error: req.URL.Path}, 404, req, res)

		return
	}

	if action[1] == "new" {
		h.Controller.processAddAccount(req, res)

		return
	}

	if detectAccount(req, res) == false {
		h.Controller.errorResponse(structs.WebError{T: "Invalid account", Error: "no such account"}, 504, req, res)

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
			h.Controller.errorResponse(structs.WebError{T: "Not found", Error: req.URL.Path}, 404, req, res)

			return
		}
		chatId, _ := strconv.ParseInt(m[1], 10, 64)
		messageId, _ := strconv.ParseInt(m[2], 10, 64)
		h.Controller.processTgSingleMessage(chatId, messageId, req, res)
		return
	case "l":
		h.Controller.processTgChatList(req, res)
		return
	case "li":
		h.Controller.processTgLink(req, res)
		return
	case "to":
		h.Controller.processTdlibOptions(req, res)
		return
	case "as":
		h.Controller.processTgActiveSessions(req, res)
		return
	case "c":
		r := regexp.MustCompile(`^/c/(-?\d+)$`)
		m := r.FindStringSubmatch(req.URL.Path)
		if m == nil {
			h.Controller.errorResponse(structs.WebError{T: "Not found", Error: req.URL.Path}, 404, req, res)

			return
		}
		chatId, _ := strconv.ParseInt(m[1], 10, 64)
		h.Controller.processTgChatInfo(chatId, req, res)

		return
	case "h":
		r := regexp.MustCompile(`^/h/?(-?\d+)?($|/)`)
		m := r.FindStringSubmatch(req.URL.Path)
		if m == nil {
			h.Controller.errorResponse(structs.WebError{T: "Not found", Error: req.URL.Path}, 404, req, res)

			return
		}
		chatId, _ := strconv.ParseInt(m[1], 10, 64)
		if m[1] == "" {
			chatId = libs.AS.Get(currentAcc).DbData.Id
		}
		ids := req.FormValue("ids")
		if ids != "" {
			h.Controller.processTgMessagesByIds(chatId, req, res)
		} else {
			h.Controller.processTgChatHistoryOnline(chatId, req, res)
		}

		return
	case "f":
		r := regexp.MustCompile(`^/f/([\w\-_]+)$`)
		m := r.FindStringSubmatch(req.URL.Path)
		if m == nil || m[1] == "" {
			h.Controller.errorResponse(structs.WebError{T: "Not found", Error: req.URL.Path}, 404, req, res)

			return
		}

		file, err := libs.AS.Get(currentAcc).TdApi.DownloadFileByRemoteId(m[1])

		if err != nil {
			h.Controller.errorResponse(structs.WebError{T: "Attachment error", Error: err.Error()}, 502, req, res)

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

		h.Controller.errorResponse(structs.WebError{T: "Invalid file", Error: file.Extra}, 504, req, res)

		return

	case "s":
		h.Controller.processSettings(req, res)
		return
	case "delete":
		r := regexp.MustCompile(`^/delete/(-?\d+)$`)
		m := r.FindStringSubmatch(req.URL.Path)
		if m == nil {
			h.Controller.errorResponse(structs.WebError{T: "Not found", Error: req.URL.Path}, 404, req, res)

			return
		}

		chatId, _ := strconv.ParseInt(m[1], 10, 64)

		h.Controller.processTgDelete(chatId, req, res)

		return
	default:
		h.Controller.errorResponse(structs.WebError{T: "Not found", Error: req.URL.Path}, 404, req, res)

		return
	}
}

func tryFile(req *http.Request, w http.ResponseWriter) bool {
	i := strings.Index(req.URL.Path, "/web/")
	var path string
	if i == -1 {
		path = "web/" + req.URL.Path
	} else if i == 0 {
		path = req.URL.Path[1:]
	} else {
		w.WriteHeader(404)
		w.Write([]byte("not found"))

		return true
	}
	stat, err := os.Stat(path)
	if err == nil && !stat.IsDir() {
		http.ServeFile(w, req, path)

		return true
	}

	return false
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

		return false
	}

	if libs.AS.Get(currentAcc) == nil {

		return false
	}

	cookie := http.Cookie{Name: "acc", Value: strconv.FormatInt(currentAcc, 10), Path: "/"}
	http.SetCookie(res, &cookie)

	return true
}
