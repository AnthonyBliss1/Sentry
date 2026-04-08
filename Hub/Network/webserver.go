package network

import (
	"html/template"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi"
)

// Web Server Handlers
// ~~~~~~~~~~~~~~~~~~~

type WatchPageData struct {
	Title       string
	DeviceID    string
	PlaylistURL string
}

func WatchHandler() http.HandlerFunc {
	tmpl := template.Must(template.ParseFiles("templates/watch.html"))

	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "deviceID")
		if deviceID == "" {
			http.Error(w, "missing deviceID", http.StatusBadRequest)
			return
		}

		data := WatchPageData{
			Title:       "Camera " + deviceID,
			DeviceID:    deviceID,
			PlaylistURL: "/hls/" + deviceID + "/stream.m3u8",
		}

		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func HLSFileServer(root string) http.Handler {
	fs := http.FileServer(http.Dir(root))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/hls/") {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, "/hls")
		}

		switch filepath.Ext(r.URL.Path) {
		case ".m3u8":
			w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
		case ".ts":
			w.Header().Set("Content-Type", "video/mp2t")
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
		}

		fs.ServeHTTP(w, r)
	})
}
