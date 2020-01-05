package main

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"
)

type Bot struct {
	wg            *sync.WaitGroup
	server        Server
	redditGen     RedditGenerator
	voiceGen      VoiceGenerator
	screenshotGen ScreenshotGenerator
	splicer       Splicer
}

func NewBot(config Config, secrets Secrets) Bot {
	server := Server{
		port:         config.Server.Port,
		templatePath: "template.html",
		data:         Data{},
	}

	redditGen, err := NewRedditGenerator(secrets, 5*time.Second, config.Subreddits)
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	return Bot{
		wg:        &wg,
		server:    server,
		redditGen: *redditGen,
		voiceGen: VoiceGenerator{
			wg:     &wg,
			Client: http.DefaultClient,
			Input:  make(chan Data),
			path:   PATH_VOICE_CLIPS,
		},
		screenshotGen: ScreenshotGenerator{
			wg:         &wg,
			Input:      make(chan Data),
			path:       PATH_SCREEN_SHOTS,
			serverAddr: "http://127.0.0.1:" + server.port,
		},
		splicer: Splicer{
			Input:          make(chan Data),
			screenshotPath: PATH_SCREEN_SHOTS,
			voiceClipPath:  PATH_VOICE_CLIPS,
			outputPath:     PATH_SPLICED,
			finishedPath:   config.FinishedFilePath,
		},
	}
}

func (bot *Bot) Start(ctx context.Context) {
	go bot.server.Start(ctx)
	go bot.voiceGen.Start(ctx)
	go bot.screenshotGen.Start(ctx)
	go bot.redditGen.Start(ctx)
	go bot.splicer.Start(ctx)

	for {
		select {
		case data := <-bot.redditGen.Output:
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
}
