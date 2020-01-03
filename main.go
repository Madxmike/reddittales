package main

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

const (
	API_ENDPOINT     = "https://www.voicery.com/api/generate"
	PATH_TALES_JSON  = "tales/"
	PATH_VOICE_CLIPS = "C:/Users/Michael/Desktop/Tales/Recordings/"
)

type TextData struct {
	Title   string `json:"title"`
	Text    string `json:"text"`
	Speaker string `json:"speaker"`
	Style   string `json:"style"`
	SSML    string `json:"ssml"`
}

func main() {
	data, err := loadAllData(PATH_TALES_JSON)
	if err != nil {
		log.Fatal(err)
	}

	for name, d := range data {
		trimmedName := strings.Trim(name, ".json")
		if VoiceClipExists(PATH_VOICE_CLIPS, trimmedName) {
			log.Printf("Skipping %s as it already exists!", trimmedName)
			continue
		}
		b, err := GetVoiceClip(http.DefaultClient, d)
		if err != nil {
			log.Fatal(err)
		}
		err = SaveVoiceFile(PATH_VOICE_CLIPS, trimmedName, b)

		b, err = ProcessVoiceRequest(http.DefaultClient, d.Title, d.Speaker, d.Style, d.SSML)
		if err != nil {
			log.Println(err)
			continue
		}
		err = SaveVoiceFile(PATH_VOICE_CLIPS, trimmedName+"_title", b)

		log.Println(err)
	}
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

func loadAllData(path string) (map[string]TextData, error) {
	fileInfos, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, errors.Wrap(err, "could not load data dir")
	}

	allData := make(map[string]TextData, len(fileInfos))
	for _, info := range fileInfos {
		if info.IsDir() {
			continue
		}
		data, err := loadData(path + info.Name())
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("could not load data for file (%s)", info.Name()))
		}

		allData[info.Name()] = data
	}

	return allData, nil
}

func loadData(fileName string) (data TextData, err error) {
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
