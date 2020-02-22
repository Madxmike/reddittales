package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"html/template"
	"log"
	"net/http"
)

func StartServer(port string) {
	r := mux.NewRouter()
	r.Handle("/", r.NotFoundHandler)
	h, err := newTemplateHandler("./templates/*")
	if err != nil {
		panic(errors.Wrap(err, "could not start server"))
	}
	r.Handle("/screenshot", h)
	server := http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: r,
	}

	panic(server.ListenAndServe())
}

type TemplateHandler struct {
	t *template.Template
}

func newTemplateHandler(templatePath string) (*TemplateHandler, error) {
	t, err := template.ParseGlob(templatePath)
	if err != nil {
		return nil, errors.Wrap(err, "could not load templates")
	}

	return &TemplateHandler{t: t}, nil
}

func (h *TemplateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	renderType := query.Get("render")
	log.Println(query)
	err := h.t.ExecuteTemplate(w, renderType, query)
	if err != nil {
		log.Println(errors.Wrap(err, "could not serve template"))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	return
}
