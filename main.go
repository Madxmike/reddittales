package main

import (
	"context"
	"flag"
	"github.com/pkg/errors"
	"log"
	"os"
	"os/signal"
	"time"
)

var (
	AgentFile = flag.String("agentFile", "", "The filepath of the agent file")
)

func main() {
	flag.Parse()
	if AgentFile == nil {
		panic(errors.New("agent file not provided"))
	}

	rw, err := NewRedditWorker(*AgentFile)
	if err != nil {
		panic(errors.Wrap(err, "could not create reddit worker"))
	}

	go StartServer(os.Getenv("PORT"))

	posts, err := rw.ScrapePosts("askreddit", "top", "day", 3)
	if err != nil {
		panic(errors.Wrap(err, "could not scrape posts"))
	}

	finished := make(chan []byte, 0)
	for _, p := range posts {
		comments, err := rw.GetComments(p, 15, FilterDistinguished, FilterKarma(1000))
		if err != nil {
			log.Println(errors.Wrap(err, "could not retrieve post comments"))
			continue
		}
		vw := newVideoWorker(p, comments)
		go vw.Process(finished)
	}

	go func() {
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Minute)
		select {
		case <-ctx.Done():
			return
		case final := <-finished:
			log.Println(final)
			return
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}
