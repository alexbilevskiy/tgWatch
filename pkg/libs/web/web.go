package web

import (
	"github.com/alexbilevskiy/tgWatch/pkg/config"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"net/http"
	"strings"
)

var verbose bool = false
var currentAcc int64

func InitWeb(webHandler *HttpHandler, grpcHandler *grpc.Server) {
	m := MultiplexerHandler{
		WebHandler:  webHandler,
		GrpcHandler: grpcHandler,
	}
	server := &http.Server{
		Addr:    config.Config.WebListen,
		Handler: h2c.NewHandler(m, &http2.Server{}), //https://kennethjenkins.net/posts/go-nginx-grpc/
	}
	go server.ListenAndServe()
}

type MultiplexerHandler struct {
	WebHandler  *HttpHandler
	GrpcHandler *grpc.Server
}

func (m MultiplexerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(
		r.Header.Get("Content-Type"), "application/grpc") {
		m.GrpcHandler.ServeHTTP(w, r)
	} else {
		m.WebHandler.ServeHTTP(w, r)
	}
}
