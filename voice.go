package main

import (
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
)

func GenerateAllVoiceClips(data map[string]Data, force bool) error {
	log.Println("Generating Voice clips")
	for name, d := range data {
		err := GenerateVoiceClip(name, d)
		if err != nil {
			log.Println(err)
			continue
		}

		b, err := ProcessVoiceRequest(http.DefaultClient, d.Title)
		if err != nil {
			return errors.Wrapf(err, "could not process %s", name)
		}
		err = SaveVoiceFile(PATH_VOICE_CLIPS, name+"_title", b)
	}
	log.Println("Finished Generating Voice Clips")
	return nil
}

func GenerateVoiceClip(name string, data Data) error {
	d := data
	d.Text = ""
	for k, text := range SplitText(data.Text) {
		d.Text = text
		b, err := ProcessVoiceRequest(http.DefaultClient, d.Text)
		if err != nil {
			return errors.Wrap(err, "could not process text")
		}
		err = SaveVoiceFile(PATH_VOICE_CLIPS, name+"_"+strconv.Itoa(k), b)
		if err != nil {
			return errors.Wrap(err, "could not save voice clip")
		}
	}

	return nil
}

func VoiceClipExists(path string, name string) bool {
	_, err := os.Open(path + name + ".mp3")
	return err == nil
}

func ProcessVoiceRequest(client *http.Client, text string) ([]byte, error) {
	req, err := http.NewRequest("GET", API_ENDPOINT, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create request")
	}

	query := req.URL.Query()
	query.Add("text", text)
	query.Add("speaker", "steven")
	query.Add("style", "narration")
	query.Add("ssml", "false")

	req.URL.RawQuery = query.Encode()
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "unable to request voice clip\n")
	}
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
