package web

import (
	"net/http"
	"strconv"

	"github.com/alexbilevskiy/tgWatch/internal/account"
	"github.com/alexbilevskiy/tgWatch/internal/config"
	"github.com/alexbilevskiy/tgWatch/internal/tdlib"
)

func Run(cfg *config.Config, astore *account.AccountsStore, creator *tdlib.AccountCreator) error {

	controller := newWebController(cfg, astore, creator)
	asm := NewAccountSelectorMiddleware(astore)

	mux := http.NewServeMux()

	mux.Handle("/{$}", asm.middleware(false, http.HandlerFunc(controller.processRoot)))

	mux.Handle("/", asm.middleware(false, http.HandlerFunc(controller.catchAll)))

	mux.Handle("/to", asm.middleware(true, http.HandlerFunc(controller.processTdlibOptions)))
	mux.Handle("/as", asm.middleware(true, http.HandlerFunc(controller.processTgActiveSessions)))
	mux.Handle("/m/{chat_id}/{message_id}", asm.middleware(true, http.HandlerFunc(controller.processTgSingleMessage)))
	mux.Handle("/h", asm.middleware(true, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		controller.processTgChatHistoryOnline(r.Context().Value("current_acc").(*account.Account).DbData.Id, r, w)
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
		Handler: logging(mux),
	}
	return server.ListenAndServe()
}
