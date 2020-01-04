package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
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

func main() {
	log.SetPrefix("[Tales] ")
	data, err := loadAllData(PATH_TALES_JSON)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	bot := NewBot()
	go bot.Start(ctx)

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
