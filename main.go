package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	sshot "github.com/slotix/pageres-go-wrapper"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
)

const (
	API_ENDPOINT      = "https://www.voicery.com/api/generate"
	PATH_TALES_JSON   = "tales/"
	PATH_VOICE_CLIPS  = "voiceclips/"
	PATH_SCREEN_SHOTS = "shots/"
)

type Data struct {
	ID       string `json:"id,omitempty"`
	Username string `json:"username"`
	Score    int    `json:"score"`
	Title    string `json:"title"`
	Text     string `json:"text"`
}

type Bot struct {
	wg            *sync.WaitGroup
	server        Server
	voiceGen      VoiceGenerator
	screenshotGen ScreenshotGenerator
}

func NewBot() Bot {
	server := Server{
		port:         "3000",
		templatePath: "template.html",
		Input:        make(chan Data),
		data:         Data{},
	}
	var wg sync.WaitGroup
	return Bot{
		wg:     &wg,
		server: server,
		voiceGen: VoiceGenerator{
			wg:       &wg,
			Client:   http.DefaultClient,
			Input:    make(chan Data),
			FileType: ".mp3",
			Path:     PATH_VOICE_CLIPS,
		},
		screenshotGen: ScreenshotGenerator{
			wg:    &wg,
			Input: make(chan Data),
			path:  PATH_SCREEN_SHOTS,
			params: sshot.Parameters{
				Command: "pageres",
				Sizes:   "1024x768",
				Crop:    "--crop",
				Scale:   "--scale 0.9",
				Timeout: "--timeout 30",
			},
			serverAddr:   "http://127.0.0.1:" + server.port,
			serverUpload: server.Input,
		},
	}
}

func (bot *Bot) Start(ctx context.Context) {
	go bot.server.Start(ctx)
	go bot.voiceGen.Start(ctx)
	go bot.screenshotGen.Start(ctx)
}

func (bot *Bot) Process(data Data) {
	log.Printf("Processing %s\n", data.ID)

	bot.wg.Add(2)
	bot.voiceGen.Input <- data
	bot.screenshotGen.Input <- data
	//TODO - Splitter
	bot.wg.Wait()
	log.Println("Both done")
}

func main() {
	log.SetPrefix("[Tales] ")
	data, err := loadAllData(PATH_TALES_JSON)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	bot := NewBot()
	bot.Start(ctx)

	for _, d := range data {
		bot.Process(d)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}

func SplitText(text string) []string {
	split := strings.SplitAfter(text, ".")
	n := 0
	for _, s := range split {
		if len(s) > 0 {
			split[n] = s
			n++
		}
	}
	split = split[:n]
	return split
}

func loadAllData(path string) ([]Data, error) {
	fileInfos, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, errors.Wrap(err, "could not load data dir")
	}

	allData := make([]Data, 0)
	for _, info := range fileInfos {
		if info.IsDir() {
			continue
		}
		data, err := loadData(path + info.Name())
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("could not load data for file (%s)", info.Name()))
		}

		if data.ID == "" {
			data.ID = strings.TrimSuffix(info.Name(), ".json")
		}
		allData = append(allData, data)
	}

	return allData, nil
}

func loadData(fileName string) (data Data, err error) {
	file, err := os.Open(fileName)
	if err != nil {
		return
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(&data)
	if err != nil {
		return
	}

	return
}
