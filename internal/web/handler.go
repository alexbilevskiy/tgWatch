package web

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/alexbilevskiy/tgWatch/internal/account"
)

func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		err := req.ParseForm()
		if err != nil {
			errorResponse(WebError{T: "Unknown error", Error: err.Error()}, http.StatusInternalServerError, req, w)
			return
		}
		ctx := req.Context()

		verbose := false
		if req.FormValue("a") == "1" {
			verbose = true
		}
		newCtx := context.WithValue(ctx, "verbose", verbose)
		var currentAcc int64
		if currentAcc, err = detectAccount(req, w); err != nil {
			errorResponse(WebError{T: "Invalid account", Error: err.Error()}, http.StatusInternalServerError, req, w)
			return
		}
		newCtx = context.WithValue(newCtx, "current_acc", currentAcc)
		newReq := req.WithContext(newCtx)

		next.ServeHTTP(w, newReq)
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

func detectAccount(req *http.Request, res http.ResponseWriter) (int64, error) {
	var currentAcc int64
	accCookie, err := req.Cookie("acc")
	if err != nil {
		log.Printf("Cookie errror: %s", err.Error())

		renderTemplates(req, res, nil, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/account_select.gohtml`)

		return -1, errors.New("missing cookie `acc`")
	}
	currentAcc, err = strconv.ParseInt(accCookie.Value, 10, 64)
	if err != nil {

		return -1, errors.New("failed to convert cookie value")
	}

	if account.AS.Get(currentAcc) == nil {
		return -1, errors.New("account from cookie does not exist")
	}

	cookie := http.Cookie{Name: "acc", Value: strconv.FormatInt(currentAcc, 10), Path: "/"}
	http.SetCookie(res, &cookie)

	return currentAcc, nil
}

func errorResponse(error WebError, code int, req *http.Request, w http.ResponseWriter) {
	w.WriteHeader(code)
	renderTemplates(req, w, error, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/error.gohtml`)
}
