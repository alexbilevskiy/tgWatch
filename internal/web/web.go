package web

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/alexbilevskiy/tgwatch/internal/account"
	"github.com/alexbilevskiy/tgwatch/internal/config"
	pbapi "github.com/alexbilevskiy/tgwatch/internal/generated/pb/api"
	"github.com/alexbilevskiy/tgwatch/internal/rpc"
	"github.com/alexbilevskiy/tgwatch/internal/tdlib"
)

func Run(log *slog.Logger, cfg *config.Config, astore *account.AccountsStore, creator *tdlib.AccountCreator) error {

	controller := newWebController(log, cfg, astore, creator)
	asm := NewAccountSelectorMiddleware(astore)

	mux := http.NewServeMux()
	grpcHandler := rpc.NewHandler(astore)
	grpcServer := grpc.NewServer()
	pbapi.RegisterTgwatchServiceServer(grpcServer, grpcHandler)
	reflection.Register(grpcServer)

	mux.Handle("/{$}", asm.middleware(false, http.HandlerFunc(controller.processRoot)))

	mux.Handle("/", asm.middleware(false, http.HandlerFunc(controller.catchAll)))

	mux.Handle("/to", asm.middleware(true, http.HandlerFunc(controller.processTdlibOptions)))
	mux.Handle("/as", asm.middleware(true, http.HandlerFunc(controller.processTgActiveSessions)))
	mux.Handle("/m/{chat_id}/{message_id}", asm.middleware(true, http.HandlerFunc(controller.processTgSingleMessage)))
	mux.Handle("/h", asm.middleware(true, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		controller.processTgChatHistoryOnline(r.Context().Value("current_acc").(*account.Account).Me.Id, r, w)
	})))
	mux.Handle("/h/{chat_id}", asm.middleware(true, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		chatId, _ := strconv.ParseInt(req.PathValue("chat_id"), 10, 64)
		ids := req.FormValue("ids")
		if ids != "" {
			controller.processTgMessagesByIds(chatId, req, w)
		} else {
			controller.processTgChatHistoryOnline(chatId, req, w)
		}
	})))
	mux.Handle("/l", asm.middleware(true, http.HandlerFunc(controller.processTgChatList)))
	mux.Handle("/li", asm.middleware(true, http.HandlerFunc(controller.processTgLink)))
	mux.Handle("/c/{chat_id}", asm.middleware(true, http.HandlerFunc(controller.processTgChatInfo)))
	mux.Handle("/f/{file_id}", asm.middleware(true, http.HandlerFunc(controller.processFile)))
	mux.Handle("/e/{emoji_id}", asm.middleware(true, http.HandlerFunc(controller.processCustomEmoji)))
	mux.Handle("/delete/{chat_id}", asm.middleware(true, http.HandlerFunc(controller.processTgDelete)))
	mux.Handle("/new", asm.middleware(false, http.HandlerFunc(controller.processAddAccount)))

	server := &http.Server{
		Addr:    cfg.WebListen,
		Handler: h2c.NewHandler(grpcMux(logging(log, mux), logging(log, grpcServer)), &http2.Server{}),
	}
	return server.ListenAndServe()
}

func grpcMux(mux http.Handler, grpcServer http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			mux.ServeHTTP(w, r)
		}
	})
}

func logging(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, req)
		log.Debug("served", "method", req.Method, "uri", req.RequestURI, "duration", time.Since(start))
	})
}
