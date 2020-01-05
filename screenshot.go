package main

import (
	"context"
	"fmt"
	sshot "github.com/slotix/pageres-go-wrapper"
	"log"
	"os"
	"sync"
)

type ScreenshotGenerator struct {
	wg           *sync.WaitGroup
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
			s.wg.Done()
		case <-ctx.Done():
			return
		}
	}
}

func (s *ScreenshotGenerator) generate(data Data) {
	log.Printf("Generating screenshots for %s\n", data.ID)
	_ = os.Mkdir(s.path+data.ID, os.ModeDir)
	lines := data.Lines()
	serverData := data
	serverData.Text = ""
	s.generateTitle(data)
	for k, text := range lines {
		serverData.Text += text
		s.serverUpload <- serverData
		s.params.Filename = fmt.Sprintf("--filename=%s/%d", s.path+data.ID, k)
		sshot.GetShots([]string{s.serverAddr}, s.params)
	}

	for k, comment := range data.Comments {
		comment.ID = fmt.Sprintf("%s/%d", data.ID, k)
		s.generate(comment)
	}
	log.Printf("Finished generating screenshots for %s\n", data.ID)
}

func (s *ScreenshotGenerator) generateTitle(data Data) {
	if data.Title != "" {
		serverData := data
		serverData.Text = ""
		s.serverUpload <- serverData
		s.params.Filename = fmt.Sprintf("--filename=%s/title", s.path+data.ID)
		sshot.GetShots([]string{s.serverAddr}, s.params)
	}
}
