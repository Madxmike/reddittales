package main

import (
	"flag"
	"github.com/pkg/errors"
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

	_, err := NewRedditWorker(*AgentFile)
	if err != nil {
		panic(errors.Wrap(err, "could not create reddit worker"))
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}
