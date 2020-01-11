package main

import (
	"context"
	"log"
	"net/http"
	"sync"
)

type Bot struct {
	wg            *sync.WaitGroup
	server        Server
	reddit        *RedditGenerator
	voiceGen      VoiceGenerator
	screenshotGen ScreenshotGenerator
	splicer       Splicer
}

func NewBot(config Config, secrets Secrets) Bot {
	server := Server{
		config: config.Server,
		data:   Data{},
	}

	reddit, err := NewRedditGenerator(secrets, config.Reddit)
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	return Bot{
		wg:     &wg,
		server: server,
		reddit: reddit,
		voiceGen: VoiceGenerator{
			Config: config.Voice,
			wg:     &wg,
			Client: http.DefaultClient,
			Input:  make(chan Data),
		},
		screenshotGen: ScreenshotGenerator{
			wg:         &wg,
			Input:      make(chan Data),
			serverAddr: "http://127.0.0.1:" + config.Server.Port,
		},
		splicer: Splicer{
			Config: config.Stitch,
			Input:  make(chan Data),
		},
	}
}

func (bot *Bot) Start(ctx context.Context) {
	go bot.server.Start(ctx)
	go bot.voiceGen.Start(ctx)
	go bot.screenshotGen.Start(ctx)
	go bot.reddit.Start(ctx)
	go bot.splicer.Start(ctx)

	for {
		select {
		case data := <-bot.reddit.Output:
			bot.Process(data)
		case <-ctx.Done():
			return
		}
	}
}

func (bot *Bot) Process(data Data) {
	log.Printf("Processing %s\n", data.ID)

	bot.wg.Add(2)
	bot.voiceGen.Input <- data
	bot.screenshotGen.Input <- data
	//TODO - Splitter
	bot.wg.Wait()
	bot.splicer.Input <- data
	log.Printf("Finished %s\n", data.ID)

}
