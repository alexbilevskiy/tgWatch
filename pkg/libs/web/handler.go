package web

import (
	"github.com/alexbilevskiy/tgWatch/pkg/libs"
	"github.com/alexbilevskiy/tgWatch/pkg/structs"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		err := req.ParseForm()
		if err != nil {
			errorResponse(structs.WebError{T: "Unknown error", Error: err.Error()}, 504, req, w)
			return
		}

		verbose = false
		if req.FormValue("a") == "1" {
			verbose = true
		}

		if detectAccount(req, w) == false {
			errorResponse(structs.WebError{T: "Invalid account", Error: "no such account"}, 504, req, w)
			return
		}
		next.ServeHTTP(w, req)
	})
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, req)
		log.Printf("%s %s %s", req.Method, req.RequestURI, time.Since(start))
	})
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

func errorResponse(error structs.WebError, code int, req *http.Request, w http.ResponseWriter) {
	w.WriteHeader(code)
	renderTemplates(req, w, error, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/error.gohtml`)
}
