package main

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
)

type VoiceGenerator struct {
	wg     *sync.WaitGroup
	Client *http.Client
	Input  chan Data
	Path   string
}

func (v *VoiceGenerator) Start(ctx context.Context) {
	for {
		select {
		case in := <-v.Input:
			err := v.generate(in)
			if err != nil {
				log.Println(err)
			}
			v.wg.Done()
		case <-ctx.Done():
			return
		}
	}
}

func (v *VoiceGenerator) generate(data Data) error {
	log.Printf("Generating voice clips for %s\n", data.ID)
	for k, text := range data.Lines() {
		b, err := v.processRequest(text)
		if err != nil {
			return errors.Wrap(err, "could not generate voice clips")
		}
		err = v.saveFile(data.ID, k, b)
		if err != nil {
			return errors.Wrap(err, "could not save voice clips files")
		}
	}
	log.Printf("Finshed Generating voice clips for %s\n", data.ID)
	return nil
}

func (v *VoiceGenerator) processRequest(text string) ([]byte, error) {
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
	log.Println(req.URL.String())
	resp, err := v.Client.Do(req)
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

func (v *VoiceGenerator) saveFile(name string, n int, b []byte) error {
	_ = os.Mkdir(v.Path+name, os.ModeDir)
	fileName := fmt.Sprintf("%s%s/%d.mp3", v.Path, name, n)
	file, err := os.Create(fileName)
	if err != nil {
		return errors.Wrapf(err, "could not create %s_%d", name, n)
	}
	defer file.Close()
	_, err = file.Write(b)
	if err != nil {
		return errors.Wrapf(err, "could not write %s_%d", name, n)
	}
	return nil
}
