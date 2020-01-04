package main

import (
	"context"
	"fmt"
	sshot "github.com/slotix/pageres-go-wrapper"
	"os"
)

type ScreenshotGenerator struct {
	Input        chan Data
	path         string
	params       sshot.Parameters
	serverAddr   string
	serverUpload chan<- Data
}

func (s *ScreenshotGenerator) Start(ctx context.Context) {
	for {
		select {
		case in := <-s.Input:
			s.generate(in)
		case <-ctx.Done():
			return
		}
	}
}

func (s *ScreenshotGenerator) generate(data Data) {
	_ = os.Mkdir(s.path+data.ID, os.ModeDir)
	serverData := data
	serverData.Text = ""
	for k, text := range SplitText(data.Text) {
		serverData.Text += text
		s.serverUpload <- serverData
		s.params.Filename = fmt.Sprintf("--filename=%s/%d", s.path+data.ID, k)

		sshot.GetShots([]string{s.serverAddr}, s.params)
	}
}
