package libs

import (
	"go-tdlib/client"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"tgWatch/structs"
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

	m := regexp.MustCompile(`^/([a-z]*?)(?:$|/.+$)`).FindStringSubmatch(req.URL.Path)
	if m == nil {
		errorResponse(structs.WebError{T: "Not found", Error: req.URL.Path}, 404, req, res)

		return
	}
	action := m[1]

	switch action {
	case "":
		renderTemplates(res, nil, `templates/base.tmpl`, `templates/navbar.tmpl`, `templates/index.tmpl`)
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
		processTgSingleMessage(chatId, messageId, res)
		return
	case "j":
		processTgJournal(req, res)
		return
	case "l":
		processTgChatList(req, res)
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
		processTgChatInfo(chatId, res)

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
			chatId = int64(me.Id)
		}

		ids := req.FormValue("ids")
		if ids != "" {
			processTgMessagesByIds(chatId, req, res)
		} else {
			processTgChatHistory(chatId, req, res)
		}

		return
	case "f":
		r := regexp.MustCompile(`^/f/(?:(\d+)|([\w\-_]+))$`)
		m := r.FindStringSubmatch(req.URL.Path)
		var file *client.File
		var err error
		if m == nil {
			errorResponse(structs.WebError{T: "Not found", Error: req.URL.Path}, 404, req, res)

			return
		}

		if m[1] != "" {
			imageId, _ := strconv.ParseInt(m[2], 10, 32)
			file, err = DownloadFile(int32(imageId))
		} else if m[2] != "" {
			file, err = DownloadFileByRemoteId(m[2])
		} else {
			errorResponse(structs.WebError{T: "Not found", Error: req.URL.Path}, 404, req, res)

			return
		}
		if err != nil {
			errorResponse(structs.WebError{T: "Attachment error", Error: err.Error()}, 502, req, res)

			return
		}
		if verbose {
			renderTemplates(res, file)

			return
		}
		if file.Local.Path != "" {
			//res.Header().Add("Content-Type", "file/jpeg")
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
