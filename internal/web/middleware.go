package web

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/alexbilevskiy/tgWatch/internal/account"
)

type AccountSelectorMiddleware struct {
	as *account.AccountsStore
}

func NewAccountSelectorMiddleware(as *account.AccountsStore) *AccountSelectorMiddleware {
	return &AccountSelectorMiddleware{as: as}
}

func (as *AccountSelectorMiddleware) middleware(requireAccount bool, next http.Handler) http.Handler {
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
		req = req.WithContext(newCtx)
		var currentAcc *account.Account
		var currentAccId int64
		if currentAccId, err = as.detectAccount(req, w); err != nil && requireAccount {
			errorResponse(WebError{T: "Invalid account", Error: err.Error()}, http.StatusInternalServerError, req, w)
			return
		}
		currentAcc = as.as.Get(currentAccId)
		if currentAcc == nil && requireAccount {
			errorResponse(WebError{T: "Missing account", Error: "no acc in store"}, http.StatusInternalServerError, req, w)
			return
		}
		newCtx = context.WithValue(newCtx, "current_acc", currentAcc)
		newCtx = context.WithValue(newCtx, "accounts_store", as.as)
		req = req.WithContext(newCtx)

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

func (as *AccountSelectorMiddleware) detectAccount(req *http.Request, res http.ResponseWriter) (int64, error) {
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

	cookie := http.Cookie{Name: "acc", Value: strconv.FormatInt(currentAcc, 10), Path: "/"}
	http.SetCookie(res, &cookie)

	return currentAcc, nil
}
