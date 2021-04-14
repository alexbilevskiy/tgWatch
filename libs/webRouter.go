package libs

import (
	"encoding/json"
	"fmt"
	"go-tdlib/client"
	"html/template"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"tgWatch/structs"
)

type HttpHandler struct{}
func (h HttpHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if req.URL.Path == "/" {
		t, errParse := template.New(`base.tmpl`).ParseFiles(`templates/base.tmpl`, `templates/navbar.tmpl`, `templates/index.tmpl`)
		if errParse != nil {
			req.URL.Path = "index.html"
		} else {
			t.Execute(res, structs.Index{T: "Hello, gopher"})
			return
		}
	}
	path := "web/" + req.URL.Path
	stat, err := os.Stat(path)
	if err == nil && !stat.IsDir() {
		http.ServeFile(res, req, path)

		return
	}

	log.Printf("HTTP: %s", req.URL.Path)
	r := regexp.MustCompile(`^/([a-z]+?)($|/.+$)`)

	m := r.FindStringSubmatch(req.URL.Path)
	if m == nil {
		res.WriteHeader(404)
		res.Write([]byte("not found "+ req.URL.Path))

		return
	}
	data := []byte(fmt.Sprintf("Request URL: %s", req.RequestURI))

	action := m[1]
	req.ParseForm()
	if req.FormValue("a") == "1" {
		verbose = true
	} else {
		verbose = false
	}
	limit := int64(50)
	if req.FormValue("limit") != "" {
		limit, _ = strconv.ParseInt(req.FormValue("limit"), 10, 64)
	}
	offset := int64(0)
	if req.FormValue("offset") != "" {
		offset, _ = strconv.ParseInt(req.FormValue("offset"), 10, 64)
	}

	switch action {
	case "e":
		r := regexp.MustCompile(`^/e/(-?\d+)/(\d+)$`)
		m := r.FindStringSubmatch(req.URL.Path)
		if m == nil {
			data := []byte(fmt.Sprintf("Unknown path %s %s", action, req.URL.Path))
			res.Write(data)

			return
		}
		chatId, _ := strconv.ParseInt(m[1], 10, 64)
		messageId, _ := strconv.ParseInt(m[2], 10, 64)
		data = []byte(processTgEdit(chatId, messageId))
		break
	case "m":
		r := regexp.MustCompile(`^/m/(-?\d+)/(\d+)$`)
		m := r.FindStringSubmatch(req.URL.Path)
		if m == nil {
			data := []byte(fmt.Sprintf("Unknown path %s %s", action, req.URL.Path))
			res.Write(data)

			return
		}
		chatId, _ := strconv.ParseInt(m[1], 10, 64)
		messageId, _ := strconv.ParseInt(m[2], 10, 64)
		processSingleMessage(chatId, messageId, res)
		return
	case "d":
		r := regexp.MustCompile(`^/d/(-?\d+)/([\d,]+)$`)
		m := r.FindStringSubmatch(req.URL.Path)
		if m == nil {
			data := []byte(fmt.Sprintf("Unknown path %s %s", action, req.URL.Path))
			res.Write(data)

			return
		}
		chatId, _ := strconv.ParseInt(m[1], 10, 64)
		messageIds := m[2]
		processTgDeleted(chatId, ExplodeInt(messageIds), res)
		return
	case "j":
		processTgJournal(limit, res)
		return
	case "l":
		refresh := false
		if req.FormValue("refresh") == "1" {
			refresh = true
		}
		var folder int32 = ClMain
		if req.FormValue("folder") != "" {
			folder64, _ := strconv.ParseInt(req.FormValue("folder"), 10, 32)
			folder = int32(folder64)
		}
		processTgChatList(refresh, folder, res)
		return
	case "o":
		processTgOverview(limit, res)
		return
	case "c":
		r := regexp.MustCompile(`^/c/(-?\d+)$`)
		m := r.FindStringSubmatch(req.URL.Path)
		if m == nil {
			data := []byte(fmt.Sprintf("Unknown path %s %s", action, req.URL.Path))
			res.Write(data)

			return
		}
		chatId, _ := strconv.ParseInt(m[1], 10, 64)
		processTgChatInfo(chatId, res)

		return
	case "h":
		r := regexp.MustCompile(`^/h/(-?\d+)$`)
		m := r.FindStringSubmatch(req.URL.Path)
		if m == nil {
			data := []byte(fmt.Sprintf("Unknown path %s %s", action, req.URL.Path))
			res.Write(data)

			return
		}

		chatId, _ := strconv.ParseInt(m[1], 10, 64)
		processTgChatHistory(chatId, limit, offset, res)

		return
	case "f":
		r := regexp.MustCompile(`^/f/((\d+)|([\w\-_]+))$`)
		m := r.FindStringSubmatch(req.URL.Path)
		var file *client.File
		var err error
		if m == nil {
			data := []byte(fmt.Sprintf("Unknown path %s %s", action, req.URL.Path))
			res.Write(data)

			return
		} else if m[2] != "" {
			imageId, _ := strconv.ParseInt(m[2], 10, 32)
			file, err = DownloadFile(int32(imageId))
		} else if m[3] != "" {
			file, err = DownloadFileByRemoteId(m[3])
		} else {
			data := []byte(fmt.Sprintf("Unknown file name %s %s", action, req.URL.Path))
			res.Write(data)

			return
		}
		if err != nil {
			errMsg := structs.MessageAttachmentError{T:"attachmentError", Id: m[1], Error: err.Error()}
			j, _ := json.Marshal(errMsg)
			data = j

			break
		}
		if file.Local.Path != "" && !verbose {
			//res.Header().Add("Content-Type", "file/jpeg")
			http.ServeFile(res, req, file.Local.Path)

			return
		}
		j, _ := json.Marshal(file)
		data = j

		break
	case "delete":
		r := regexp.MustCompile(`^/delete/(-?\d+)$`)
		m := r.FindStringSubmatch(req.URL.Path)
		if m == nil {
			data := []byte(fmt.Sprintf("Unknown path %s %s", action, req.URL.Path))
			res.Write(data)

			return
		}

		chatId, _ := strconv.ParseInt(m[1], 10, 64)
		pattern := req.FormValue("pattern")
		if pattern == "" || len(pattern) < 3 {
			data := []byte(fmt.Sprintf("Unknown pattern `%s`", pattern))
			res.Write(data)

			return
		}
		limit := 50
		if req.FormValue("limit") != "" {
			limit64, _ := strconv.ParseInt(req.FormValue("limit"), 10, 0)
			limit = int(limit64)
		}

		processTgDelete(chatId, pattern, limit, res)

		return
	default:
		res.WriteHeader(404)
		res.Write([]byte("not found " + req.URL.Path))

		return
	}

	res.WriteHeader(200)
	res.Write(data)
}
