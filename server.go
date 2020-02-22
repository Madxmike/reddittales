package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
)

func StartServer(port string) {
	r := mux.NewRouter()
	r.Handle("/", r.NotFoundHandler)
	r.HandleFunc("/screenshot", ServeScreenshotPage)
	server := http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: r,
	}

	panic(server.ListenAndServe())
}

func ServeScreenshotPage(w http.ResponseWriter, r *http.Request) {

}
