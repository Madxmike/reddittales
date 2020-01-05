package main

import (
	"context"
	"github.com/pkg/errors"
	stripmd "github.com/writeas/go-strip-markdown"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"
)

type Server struct {
	port         string
	templatePath string
	Input        chan Data
	data         Data
	temp         *template.Template
}

func (server *Server) Start(ctx context.Context) {
	t, err := template.ParseGlob(server.templatePath)
	if err != nil {
		panic(err)
	}
	server.temp = t

	go func() {
		for {
			select {
			case data := <-server.Input:
				server.data = data
			case <-ctx.Done():
				return
			}
			server.data = <-server.Input
		}
	}()

	err = http.ListenAndServe(":"+server.port, server)
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
	server.data.Text = stripmd.Strip(server.data.Text)
	server.data.Text = strings.TrimPrefix(server.data.Text, server.data.Title)
	err := server.temp.ExecuteTemplate(w, "index", server.data)
	if err != nil {
		return errors.Wrap(err, "could not execute template file")
	}

	return nil
}
