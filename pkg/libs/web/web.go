package web

import (
	"github.com/alexbilevskiy/tgWatch/pkg/config"
	"github.com/alexbilevskiy/tgWatch/pkg/libs"
	"net/http"
	"strconv"
)

func InitWeb() {

	controller := webController{}

	mux := http.NewServeMux()

	mux.Handle("/{$}", middleware(http.HandlerFunc(controller.processRoot)))

	mux.HandleFunc("/", controller.catchAll)

	mux.Handle("/to", middleware(http.HandlerFunc(controller.processTdlibOptions)))
	mux.Handle("/as", middleware(http.HandlerFunc(controller.processTgActiveSessions)))
	mux.Handle("/m/{chat_id}/{message_id}", middleware(http.HandlerFunc(controller.processTgSingleMessage)))
	mux.Handle("/h", middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		controller.processTgChatHistoryOnline(libs.AS.Get(r.Context().Value("current_acc").(int64)).DbData.Id, r, w)
	})))
	mux.Handle("/h/{chat_id}", middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		chatId, _ := strconv.ParseInt(req.PathValue("chat_id"), 10, 64)
		ids := req.FormValue("ids")
		if ids != "" {
			controller.processTgMessagesByIds(chatId, req, w)
		} else {
			controller.processTgChatHistoryOnline(chatId, req, w)
		}
	})))
	mux.Handle("/l", middleware(http.HandlerFunc(controller.processTgChatList)))
	mux.Handle("/li", middleware(http.HandlerFunc(controller.processTgLink)))
	mux.Handle("/c/{chat_id}", middleware(http.HandlerFunc(controller.processTgChatInfo)))
	mux.Handle("/s", middleware(http.HandlerFunc(controller.processSettings)))
	mux.Handle("/f/{file_id}", middleware(http.HandlerFunc(controller.processFile)))
	mux.Handle("/delete/{chat_id}", middleware(http.HandlerFunc(controller.processTgDelete)))
	mux.HandleFunc("/new", controller.processAddAccount)

	server := &http.Server{
		Addr:    config.Config.WebListen,
		Handler: logging(mux),
	}
	go server.ListenAndServe()
}
