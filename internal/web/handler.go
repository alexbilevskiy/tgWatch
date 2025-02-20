package web

import (
	"net/http"
	"os"
	"strings"
)

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

func errorResponse(error WebError, code int, req *http.Request, w http.ResponseWriter) {
	w.WriteHeader(code)
	renderTemplates(req, w, error, `templates/base.gohtml`, `templates/navbar.gohtml`, `templates/error.gohtml`)
}
