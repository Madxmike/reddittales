package main

import (
	"context"
	"encoding/json"
	"github.com/jzelinskie/geddit"
	"github.com/pkg/errors"
	sshot "github.com/slotix/pageres-go-wrapper"
	"log"
	"net/http"
	"os"
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

func NewBot() Bot {
	server := Server{
		port:         "3000",
		templatePath: "template.html",
		Input:        make(chan Data),
		data:         Data{},
	}

	secrets, err := loadSecrets("secrets.json")
	if err != nil {
		panic(err)
	}
	redditGen, err := NewRedditGenerator(secrets, 5*time.Second, geddit.TopSubmissions, geddit.ListingOptions{
		Time:    geddit.ThisDay,
		Limit:   3,
		After:   "",
		Before:  "",
		Count:   0,
		Show:    "",
		Article: "",
	}, "maliciouscompliance")
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
			Path:   PATH_VOICE_CLIPS,
		},
		screenshotGen: ScreenshotGenerator{
			wg:    &wg,
			Input: make(chan Data),
			path:  PATH_SCREEN_SHOTS,
			params: sshot.Parameters{
				Command: "pageres",
				Sizes:   "1024x1080",
				Crop:    "--crop",
				Scale:   "--scale 0.9",
				Timeout: "--timeout 30",
			},
			serverAddr:   "http://127.0.0.1:" + server.port,
			serverUpload: server.Input,
		},
		splicer: Splicer{
			Input:          make(chan Data),
			screenshotPath: PATH_SCREEN_SHOTS,
			voiceClipPath:  PATH_VOICE_CLIPS,
			outputPath:     PATH_SPLICED,
		},
	}
}

func loadSecrets(filename string) (Secrets, error) {
	var secrets Secrets
	file, err := os.Open(filename)
	if err != nil {
		return secrets, errors.Wrap(err, "could not open secrets file")
	}
	defer file.Close()
	err = json.NewDecoder(file).Decode(&secrets)
	if err != nil {
		return secrets, errors.Wrap(err, "could not decode secrets file")
	}

	return secrets, nil
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
