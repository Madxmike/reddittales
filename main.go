package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
)

const (
	PATH_TALES_JSON   = "tales/"
	PATH_VOICE_CLIPS  = "voiceclips/"
	PATH_SCREEN_SHOTS = "shots/"
	PATH_SPLICED      = "spliced/"
)

var (
	OutputDir           = flag.String("output", "", "The path finished files are outputted to")
	BackgroundVideoPath = flag.String("background", "", "The path of the video to use as a background")
)

func init() {
	flag.Parse()
}

func main() {
	config, err := LoadConfig("config.json")
	if err != nil {
		panic(err)
	}
	secrets, err := LoadSecrets("secrets.json")
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	//TODO - Flags
	bot := NewBot(config, secrets)
	go bot.Start(ctx)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}
