package network

import (
	"fmt"
	"io"
	"net/http"

	utils "github.com/anthonybliss1/Sentry/Hub/Utils"
	"github.com/go-chi/chi"
)

type FileServer struct {
	*http.Server
}

// FS Handler
// ~~~~~~~~~~~~~~~~~

func FSHandler(hlsDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "deviceID") // grab the id from the qParam
		fileName := chi.URLParam(r, "fileName") // grab the fileName

		// read request body (file data)
		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to read request body: %w", err), http.StatusInternalServerError)
		}
		defer r.Body.Close()

		if err := utils.SaveFile(hlsDir, deviceID, fileName, data); err != nil {
			http.Error(w, fmt.Sprintf("Could not save file: %w", err), http.StatusInternalServerError)
		}

		msg := fmt.Sprintf("Successfully Uploaded File [%s] for device: %s", fileName, deviceID) // confirm save

		w.Write([]byte(msg)) // write msg
		w.WriteHeader(http.StatusOK)
	}
}
