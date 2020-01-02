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
	API_ENDPOINT = "https://www.voicery.com/api/generate"
)

type TextData struct {
	Text    string `json:"text"`
	Speaker string `json:"speaker"`
	Style   string `json:"style"`
	SSML    string `json:"ssml"`
}

func main() {
	data, err := loadAllData("tales/")
	log.Println(len(data))
	log.Println(err)

	for name, d := range data {
		resp, err := GetVoiceClip(http.DefaultClient, d)
		if err != nil {
			log.Fatal(err)
		}

		file, err := os.Create("voiceclips/" + strings.Trim(name, ".json") + ".mp3")
		defer file.Close()
		if err != nil {
			panic(err)
		}
		bytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		_, err = file.Write(bytes)

		log.Println(err)
	}

}

func GetVoiceClip(client *http.Client, data TextData) (*http.Response, error) {
	req, err := http.NewRequest("GET", API_ENDPOINT, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get voice clip")
	}
	query := req.URL.Query()
	query.Add("text", data.Text)
	query.Add("speaker", data.Speaker)
	query.Add("style", data.Style)
	query.Add("ssml", data.SSML)
	req.URL.RawQuery = query.Encode()
	return client.Do(req)
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
	defer file.Close()
	if err != nil {
		return
	}

	err = json.NewDecoder(file).Decode(&data)
	if err != nil {
		return
	}

	return
}
