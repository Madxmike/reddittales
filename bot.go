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

	reddit, err := NewRedditGenerator(secrets, config.Reddit, 5*time.Second)
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	return Bot{
		wg:     &wg,
		server: server,
		reddit: reddit,
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
}
