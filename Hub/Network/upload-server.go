package network

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func CreateFS() {
	r := chi.NewRouter()

	r.Use(middleware.Logger)

	r.Post("/upload/{deviceID}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "deviceID") // grab the id from the qParam

		msg := fmt.Sprintf("Uploaded File for device: %s", id) // test msg

		w.Write([]byte(msg)) // write msg
	})

	// will probably allow the user to configure the port? not sure
	// or use some obscure port
	addr := fmt.Sprintf("0.0.0.0:%d", 8080)

	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
