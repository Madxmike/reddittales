package main

import (
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"
)

type Server struct {
	config         serverConfig
	data           Data
	renderTemplate *template.Template
}

func (s *Server) Start(ctx context.Context) {
	t, err := template.ParseGlob(s.config.TemplatePath)
	if err != nil {
		panic(err)
	}
	s.renderTemplate = t
	http.Handle("/", s)
	http.HandleFunc("/push", s.postData)
	err = http.ListenAndServe(":"+s.config.Port, nil)
	if err != nil {
		panic(err)
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := s.executeTemplate(w)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *Server) postData(w http.ResponseWriter, r *http.Request) {
	var data Data
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		err = errors.Wrap(err, "could not recieve data")
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	s.data = data
	w.WriteHeader(http.StatusOK)
}

func (s *Server) executeTemplate(w io.Writer) error {
	if s.config.RefreshTemplate {
		t, err := template.New("post").ParseGlob(s.config.TemplatePath)
		if err != nil {
			return errors.Wrap(err, "could not refresh template")
		}
		s.renderTemplate = t
	}
	data := s.data
	data.Text = strings.TrimPrefix(data.Text, data.Title)
	err := s.renderTemplate.ExecuteTemplate(w, "index", data)
	if err != nil {
		return errors.Wrap(err, "could not execute template file")
	}

	return nil
}
