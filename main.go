package main

import (
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
	PATH_VOICE_CLIPS  = "C:/Users/Michael/Desktop/Tales/Recordings/"
	PATH_SCREEN_SHOTS = "shots/"
)

type Data struct {
	Username string `json:"username"`
	Score    int    `json:"score"`
	Title    string `json:"title"`
	Text     string `json:"text"`
}

func main() {
	data, err := loadAllData(PATH_TALES_JSON)
	if err != nil {
		log.Fatal(err)
	}
	//TODO - Make this passed arguments
	server := Server{
		port:         "3000",
		templatePath: "template.html",
		Render:       make(chan Data, 1),
		data:         Data{},
	}
	log.Println("test")

	go server.Start()
	server.Render <- Data{
		Username: "test",
		Score:    100,
		Title:    "title",
		Text:     "text",
	}
	log.Println("server started")

	_ = GenerateAllScreenshots(data, server.Render, PATH_SCREEN_SHOTS)

	err = GenerateAllVoiceClips(data, false)
	if err != nil {
		log.Println(err)
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}

func SplitText(text string) []string {
	split := strings.Split(text, ".")
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

func loadAllData(path string) (map[string]Data, error) {
	fileInfos, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, errors.Wrap(err, "could not load data dir")
	}

	allData := make(map[string]Data, len(fileInfos))
	for _, info := range fileInfos {
		if info.IsDir() {
			continue
		}
		data, err := loadData(path + info.Name())
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("could not load data for file (%s)", info.Name()))
		}

		name := strings.TrimSuffix(info.Name(), ".json")
		allData[name] = data
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
