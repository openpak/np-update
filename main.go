// Command np-update answers PS3 title patch list requests (/tpl/<env>/<titleid>/<titleid>-ver.xml)
// with "no update available" for every title. OpenPak never hosts Sony or game patch packages.
package main

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
)

// handler replies 404 with an empty body to every well-formed tpl request. VSH treats 403 and 404
// as "no patch list" (FUN_00281734) and raises no error.
func handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("GET /tpl/", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("title patch check", "path", r.URL.Path)
		w.Header().Set("Content-Length", "0")
		w.WriteHeader(http.StatusNotFound)
	})
	return mux
}

func main() {
	addr := os.Getenv("NP_HTTP_ADDR")
	if strings.TrimSpace(addr) == "" {
		addr = ":8534"
	}
	slog.Info("np-update listening", "addr", addr)
	if err := http.ListenAndServe(addr, handler()); err != nil {
		slog.Error("np-update stopped", "err", err)
		os.Exit(1)
	}
}
