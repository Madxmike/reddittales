package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
)

var (
	ConfigPath = flag.String("config", "", "The path of the config")
)

func init() {
	flag.Parse()
}

func main() {
	log.Println(os.TempDir())
	config, err := LoadConfig(*ConfigPath)
	if err != nil {
		panic(err)
	}
	secrets, err := LoadSecrets("secrets.json")
	if err != nil {
		panic(err)
	}

	//TODO - Flags
	bot := NewBot(config, secrets)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go bot.Start(ctx)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}
