package web

import (
	"github.com/alexbilevskiy/tgWatch/pkg/config"
	"net/http"
)

var verbose bool = false
var currentAcc int64

func InitWeb() {
	server := &http.Server{
		Addr: config.Config.WebListen,
		Handler: HttpHandler{
			Controller: webController{},
		},
	}
	go server.ListenAndServe()
}
