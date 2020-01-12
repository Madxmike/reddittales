package main

import (
	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"context"
	"fmt"
	"github.com/pkg/errors"
	texttospeechpb "google.golang.org/genproto/googleapis/cloud/texttospeech/v1"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
)

type VoiceGenerator struct {
	Config voiceConfig
	wg     *sync.WaitGroup
	Client *http.Client
	Input  chan Data
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
	ctx := context.Background()
	client, err := texttospeech.NewClient(ctx)
	if err != nil {
		return errors.Wrap(err, "could not start tts client")
	}

	dirName := fmt.Sprintf("%s%c%s%c", os.TempDir(), os.PathSeparator, data.ID, os.PathSeparator)
	_ = os.Mkdir(dirName, os.ModeDir)

	sentences := data.Sentences()
	if data.Title != "" {
		sentences = append([]string{data.Title}, sentences...)
	}

	serverData := data
	serverData.Text = ""
	for n, text := range sentences {
		b, err := v.processRequest(client, text)
		if err != nil {
			return errors.Wrap(err, "could not generate voice clips")
		}

		filename := fmt.Sprintf("%s%c%d.mp3", dirName, os.PathSeparator, n)
		err = ioutil.WriteFile(filename, b, 0777)
		if err != nil {
			return errors.Wrap(err, "could not save voice clip")
		}
	}

	for _, comment := range data.Comments {
		comment.ID = fmt.Sprintf("%s%c%s", data.ID, os.PathSeparator, comment.ID)
		_ = v.generate(comment)
	}
	return nil
}

func (v *VoiceGenerator) processRequest(client *texttospeech.Client, text string) ([]byte, error) {
	ctx := context.Background()

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
