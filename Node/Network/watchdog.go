package network

import (
	"os"
	"path/filepath"
	"strings"

	utils "github.com/anthonybliss1/Sentry/Node/Utils"
	"github.com/fsnotify/fsnotify"
)

func DeployWatchdog(n *NodeClient, hlsDir string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	// add the hlsDir to the watcher for monitoring
	if err := watcher.Add(hlsDir); err != nil {
		return err
	}

	// create playlist path to not rely on '.tmp' files
	playlistPath := filepath.Join(hlsDir, "stream.m3u8")

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			name := event.Name
			base := filepath.Base(name)

			// trigger for final .ts segements that are ready to send to Hub
			// using write because create would result in picking up imcomplete files
			if strings.HasSuffix(base, ".ts") && (event.Has(fsnotify.Create) || event.Has(fsnotify.Write)) {
				utils.Blue.Printf("<SEGMENT EVENT> %s %s\n", event.Op.String(), name)

				// upload file
				go func(filePath string) {
					if err := n.UploadFile(filePath); err != nil {
						red.Println(err) // dont want to crash the pipeline on an upload error
					}
				}(name)

				continue
			}

			// trigger for final .m3u8 playlist file this is ready to send to Hub
			if base == "stream.m3u8" && (event.Has(fsnotify.Create) || event.Has(fsnotify.Write)) {
				if _, err := os.Stat(playlistPath); err == nil {
					utils.Blue.Printf("<PLAYLIST EVENT> %s %s\n", event.Op.String(), playlistPath)

					// upload file
					go func(filePath string) {
						if err := n.UploadFile(filePath); err != nil {
							red.Println(err) // dont want to crash the pipeline on an upload error
						}
					}(playlistPath)
				}

				continue
			}

		// handle errors
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}

			red.Printf("Watchdog Error: %q\n", err)
		}
	}
}
