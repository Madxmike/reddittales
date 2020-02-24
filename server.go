package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"html/template"
	"log"
	"net/http"
	"strings"
)

func StartServer(port string) {
	r := mux.NewRouter()
	r.Handle("/", r.NotFoundHandler)
	h, err := newTemplateHandler("./templates/*")
	if err != nil {
		panic(errors.Wrap(err, "could not start server"))
	}
	r.Handle("/screenshot", h)
	r.PathPrefix("/static/").Handler(Static("/static/"))
	server := http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: r,
	}

	panic(server.ListenAndServe())
}

func Static(path string) http.Handler {
	fs := http.FileServer(http.Dir("." + path))
	fs = http.StripPrefix(path, fs)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		fs.ServeHTTP(w, r)
	})
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
	t, err := template.ParseGlob("./templates/*")
	if err != nil {
		return
	}
	h.t = t
	query := r.URL.Query()
	renderType := query.Get("render")
	data := struct {
		Author string
		Karma  string
		Text   string
	}{
		Author: query.Get("author"),
		Karma:  query.Get("karma"),
		Text:   query.Get("text"),
	}

	err = h.t.ExecuteTemplate(w, renderType, data)
	if err != nil {
		log.Println(errors.Wrap(err, "could not serve template"))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
