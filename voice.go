package main

import (
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func GenerateAllVoiceClips(data map[string]TextData, force bool) error {
	for name, d := range data {
		if !force && VoiceClipExists(PATH_VOICE_CLIPS, name) {
			log.Printf("Skipping %s as it already exists!", name)
			continue
		}
		b, err := GetVoiceClip(http.DefaultClient, d)
		if err != nil {
			return errors.Wrapf(err, "could not get %s", name)
		}
		err = SaveVoiceFile(PATH_VOICE_CLIPS, name, b)

		b, err = ProcessVoiceRequest(http.DefaultClient, d.Title, d.Speaker, d.Style, d.SSML)
		if err != nil {
			return errors.Wrapf(err, "could not process %s", name)
		}
		err = SaveVoiceFile(PATH_VOICE_CLIPS, name+"_title", b)
		log.Println(err)
	}
}

func VoiceClipExists(path string, name string) bool {
	_, err := os.Open(path + name + ".mp3")
	return err == nil
}

func GetVoiceClip(client *http.Client, data TextData) ([]byte, error) {

	bytes := make([]byte, 0)

	for _, text := range SplitText(data.Text) {
		b, err := ProcessVoiceRequest(client, text, data.Speaker, data.Style, data.SSML)
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
