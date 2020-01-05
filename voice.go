package main

import (
	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"context"
	"fmt"
	"github.com/pkg/errors"
	texttospeechpb "google.golang.org/genproto/googleapis/cloud/texttospeech/v1"
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
	lines := data.Lines()
	v.createDir(data)
	for k, text := range lines {
		b, err := v.processRequest(text)
		if err != nil {
			return errors.Wrap(err, "could not generate voice clips")
		}
		fileName := fmt.Sprintf("%s%s/%d.mp3", v.Path, data.ID, k)
		err = v.saveFile(fileName, b)
		if err != nil {
			return errors.Wrap(err, "could not save voice clips files")
		}
	}

	for k, comment := range data.Comments {
		comment.ID = fmt.Sprintf("%s/%d", data.ID, k)
		_ = v.generate(comment)
	}
	_ = v.generateTitle(data)
	log.Printf("Finshed Generating voice clips for %s\n", data.ID)
	return nil
}

func (v *VoiceGenerator) generateTitle(data Data) error {
	if data.Title == "" {
		return nil
	}
	b, err := v.processRequest(data.Title)
	if err != nil {
		return errors.Wrap(err, "could not generate title voice clip")
	}
	fileName := fmt.Sprintf("%s%s/title.mp3", v.Path, data.ID)
	err = v.saveFile(fileName, b)
	if err != nil {
		return errors.Wrap(err, "could not save title voice clip")
	}

	return nil
}

func (v *VoiceGenerator) createDir(data Data) {
	_ = os.Mkdir(v.Path+data.ID, os.ModeDir)
}

func (v *VoiceGenerator) saveFile(fileName string, b []byte) error {
	file, err := os.Create(fileName)
	if err != nil {
		return errors.Wrapf(err, "could not create %s", fileName)
	}
	defer file.Close()
	_, err = file.Write(b)
	if err != nil {
		return errors.Wrapf(err, "could not write %s", fileName)
	}
	return nil
}

func (v *VoiceGenerator) processRequest(text string) ([]byte, error) {
	ctx := context.Background()
	client, err := texttospeech.NewClient(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not start tts client")
	}

	req := texttospeechpb.SynthesizeSpeechRequest{
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{Text: text},
		},
		Voice: &texttospeechpb.VoiceSelectionParams{
			LanguageCode: "en-US",
			SsmlGender:   texttospeechpb.SsmlVoiceGender_NEUTRAL,
		},
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding: texttospeechpb.AudioEncoding_MP3,
		},
	}

	resp, err := client.SynthesizeSpeech(ctx, &req)
	if err != nil {
		return nil, errors.Wrap(err, "could not synthesize text")
	}
	return resp.AudioContent, nil
}
