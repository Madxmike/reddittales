package main

import (
	"github.com/pkg/errors"
	"html/template"
	"io"
	"log"
	"net/http"
)

type Server struct {
	port         string
	templatePath string
	data         Data
}

func (server *Server) Start() {
	err := http.ListenAndServe(":"+server.port, server)
	if err != nil {
		panic(err)
	}
}

func (server *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := server.executeTemplate(w)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (server *Server) executeTemplate(w io.Writer) error {
	t, err := template.ParseGlob(server.templatePath)
	if err != nil {
		return errors.Wrap(err, "could not parse template file")
	}
	err = t.ExecuteTemplate(w, "index", server.data)
	if err != nil {
		return errors.Wrap(err, "could not execute template file")
	}

	return nil
}
