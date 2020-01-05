package main

import (
	"context"
	"github.com/madxmike/reddittales/internal"
	"log"
	"os"
	"os/signal"
)

const (
	PATH_TALES_JSON   = "tales/"
	PATH_VOICE_CLIPS  = "voiceclips/"
	PATH_SCREEN_SHOTS = "shots/"
	PATH_SPLICED      = "spliced/"
)

func main() {
	log.SetPrefix("[Tales] ")
	data, err := internal.LoadAllData(PATH_TALES_JSON)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	//TODO - Flags
	bot := internal.NewBot(PATH_VOICE_CLIPS, PATH_SCREEN_SHOTS, PATH_SPLICED)
	go bot.Start(ctx)

	for _, d := range data {
		bot.Process(d)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}
