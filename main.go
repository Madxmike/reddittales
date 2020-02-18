package main

import (
	"flag"
	"github.com/pkg/errors"
	"log"
	"os"
	"os/signal"
)

var (
	AgentFile = flag.String("agentFile", "", "The filepath of the agent file")
)

func main() {
	flag.Parse()
	if AgentFile == nil {
		panic(errors.New("agent file not provided"))
	}

	worker, err := NewRedditWorker(*AgentFile)

	posts, err := worker.ScrapePosts("askreddit", "top", "day", 3)
	if err != nil {
		panic(err)
	}

	for _, p := range posts {
		log.Println(p.Name)
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}
