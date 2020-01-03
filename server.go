package main

import (
	"github.com/pkg/errors"
	"html/template"
	"io"
	"log"
	"net/http"
)

type Server struct {
	Port         string
	TemplatePath string
	Render       <-chan TextData
	data         TextData
}

func (server *Server) Start() {
	err := http.ListenAndServe(":"+server.Port, server)
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
	select {
	case d := <-server.Render:
		server.data = d
	default:
	}
	t, err := template.ParseGlob(server.TemplatePath)
	if err != nil {
		return errors.Wrap(err, "could not parse template file")
	}
	err = t.Execute(w, server.data)
	if err != nil {
		return errors.Wrap(err, "could not execute template file")
	}

	return nil
}
