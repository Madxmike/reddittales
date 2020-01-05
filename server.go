package main

import (
	"context"
	"encoding/json"
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
	data         Data
	temp         *template.Template
}

func (server *Server) Start(ctx context.Context) {
	t, err := template.ParseGlob(server.templatePath)
	if err != nil {
		panic(err)
	}
	server.temp = t
	http.HandleFunc("/push", server.postData)
	http.Handle("/", server)
	err = http.ListenAndServe(":"+server.port, nil)
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

func (server *Server) postData(w http.ResponseWriter, r *http.Request) {
	var data Data
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		err = errors.Wrap(err, "could not recieve data")
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	server.data = data
	w.WriteHeader(http.StatusOK)
}

func (server *Server) executeTemplate(w io.Writer) error {
	data := server.data
	data.Text = stripmd.Strip(data.Text)
	data.Text = strings.TrimPrefix(data.Text, data.Title)
	err := server.temp.ExecuteTemplate(w, "index", data)
	if err != nil {
		return errors.Wrap(err, "could not execute template file")
	}

	return nil
}
