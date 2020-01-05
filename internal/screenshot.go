package internal

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
	serverData := data
	serverData.Text = ""
	for k, text := range data.Lines() {
		s.serverUpload <- serverData
		s.params.Filename = fmt.Sprintf("--filename=%s/%d", s.path+data.ID, k)
		sshot.GetShots([]string{s.serverAddr}, s.params)
		serverData.Text += text
	}
	log.Printf("Finished generating screenshots for %s\n", data.ID)

}
