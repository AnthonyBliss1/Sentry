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

	// add the hlsDir to the watcher for monitoring
	if err := watcher.Add(hlsDir); err != nil {
		watcher.Close()
		return err
	}

	// create playlist path to not rely on '.tmp' files
	playlistPath := filepath.Join(hlsDir, "stream.m3u8")

	go func() {
		defer watcher.Close()

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				name := event.Name
				base := filepath.Base(name)

				// trigger for final .ts segements that are ready to send to Hub
				// using write because create would result in picking up imcomplete files
				if strings.HasSuffix(base, ".ts") && event.Has(fsnotify.Write) {
					utils.Blue.Printf("<SEGMENT> Sending complete segment [ %s ]\n", name)

					// upload file
					go func() {
						if err := n.UploadFile(name); err != nil {
							red.Println(err) // dont want to crash the pipeline on an upload error
							return
						}
					}()
					continue
				}

				// trigger for final .m3u8 playlist file this is ready to send to Hub
				if base == "stream.m3u8" && (event.Has(fsnotify.Create) || event.Has(fsnotify.Write)) {
					// check its a valid file (kinda pointless but need to check)
					if _, err := os.Stat(playlistPath); err != nil {
						continue
					}

					utils.Blue.Printf("<PLAYLIST> Sending complete playlist [ %s ]\n", playlistPath)

					// upload file
					go func() {
						if err := n.UploadFile(playlistPath); err != nil {
							red.Println(err) // dont want to crash the pipeline on an upload error
							return
						}
					}()
					continue
				}

			// handle errors
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}

				red.Printf("Watchdog Error: %q\n", err)
			}
		}
	}()

	// block forever
	<-make(chan struct{})

	return nil
}
