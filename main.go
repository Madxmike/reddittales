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
	PATH_VOICE_CLIPS = "voiceclips/"
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
		log.Println(err)
	}
}

func VoiceClipExists(path string, name string) bool {
	_, err := os.Open(path + name + ".mp3")
	return err == nil
}

func GetVoiceClip(client *http.Client, data TextData) ([]byte, error) {

	bytes := make([]byte, 0)
	b, err := ProcessVoiceRequest(client, data.Title, data.Speaker, data.Style, data.SSML)
	if err != nil {
		return nil, errors.Wrap(err, "could not process title")
	}
	bytes = append(bytes, b...)
	for _, text := range SplitText(data.Text) {
		b, err = ProcessVoiceRequest(client, text, data.Speaker, data.Style, data.SSML)
		if err != nil {
			return nil, errors.Wrap(err, "could not process text")
		}
		bytes = append(bytes, b...)
	}

	return bytes, nil
}

func ProcessVoiceRequest(client *http.Client, text, speaker, style, ssml string) ([]byte, error) {
	req, err := http.NewRequest("GET", API_ENDPOINT, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create request")
	}

	query := req.URL.Query()
	query.Add("text", text)
	query.Add("speaker", speaker)
	query.Add("style", style)
	query.Add("ssml", ssml)
	req.URL.RawQuery = query.Encode()
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "unable to request voice clip\n")
	}
	log.Println(resp.Status)
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read response")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("could not get voice clip")
	}
	return b, nil
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

func SaveVoiceFile(path string, name string, b []byte) error {
	fileName := path + name + ".mp3"
	file, err := os.Create(fileName)
	if err != nil {
		return errors.Wrap(err, "could not create save file")
	}
	defer file.Close()
	_, err = file.Write(b)
	if err != nil {
		return errors.Wrap(err, "could not write to save file")
	}

	return nil
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
