package main

import (
	"log"
	"mime"
	"net/http"

	"grid/internal"
)

func init() {
	mime.AddExtensionType(".js", "application/javascript")
}

func main() {
	hub := internal.NewHub()

	fs := http.FileServer(http.Dir("./public/dist"))
	http.Handle("/", fs)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		internal.HandleWS(hub, w, r)
	})

	internal.StartAutosave(hub)
	internal.SetupShutdownSave(hub)

	log.Println("Server running on :4000")

	if err := http.ListenAndServe("0.0.0.0:4000", nil); err != nil {
		log.Fatal(err)
	}
}
