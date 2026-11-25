package api

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed ui/*
var dashboardFiles embed.FS

func registerDashboard(mux *http.ServeMux) {
	assets, err := fs.Sub(dashboardFiles, "ui")
	if err != nil {
		panic(err)
	}

	mux.Handle(
		"GET /ui/",
		http.StripPrefix("/ui/", http.FileServer(http.FS(assets))),
	)

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		data, err := dashboardFiles.ReadFile("ui/index.html")
		if err != nil {
			http.Error(w, "Dashboard unavailable", 500)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})
}
