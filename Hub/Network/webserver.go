package network

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed templates/stream.html
var templateFS embed.FS

var streamTmpl = template.Must(template.ParseFS(templateFS, "templates/stream.html"))

// Web Server Handlers
// ~~~~~~~~~~~~~~~~~~~

type StreamPlayer struct {
	DeviceID    string
	Title       string
	PlaylistURL string
}

type StreamPageData struct {
	Title   string
	Players []StreamPlayer
}

func StreamHandler(hlsDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entries, err := os.ReadDir(hlsDir)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to read hls directory: %q", err), http.StatusInternalServerError)
		}

		var players []StreamPlayer

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			deviceID := entry.Name()
			playlistPath := filepath.Join(hlsDir, deviceID, "stream.m3u8")

			if _, err := os.Stat(playlistPath); err != nil {
				continue
			}

			players = append(players, StreamPlayer{
				DeviceID:    deviceID,
				Title:       deviceID,
				PlaylistURL: "/hls/" + deviceID + "/stream.m3u8",
			})
		}

		sort.Slice(players, func(i, j int) bool {
			return players[i].DeviceID < players[j].DeviceID
		})

		data := StreamPageData{
			Title:   "Sentry Dashboard",
			Players: players,
		}

		if err := streamTmpl.Execute(w, data); err != nil {
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
