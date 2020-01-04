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
	lines = append([]string{data.Title}, lines...)
	for k, text := range lines {
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
