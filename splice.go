package main

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"log"
	"os"
	"os/exec"
)

type Splicer struct {
	Input          chan Data
	screenshotPath string
	voiceClipPath  string
	outputPath     string
}

func (s *Splicer) Start(ctx context.Context) {
	for {
		select {
		case in := <-s.Input:
			err := s.splice(in)
			if err != nil {
				log.Println(err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Splicer) splice(data Data) error {
	dirName := fmt.Sprintf("%s%s/", s.outputPath, data.ID)
	_ = os.Mkdir(dirName, os.ModeDir)
	for k := range data.Lines() {
		ssName := fmt.Sprintf("%s%s/%d.png", s.screenshotPath, data.ID, k)
		voiceName := fmt.Sprintf("%s%s/%d.mp3", s.voiceClipPath, data.ID, k)
		spliceName := fmt.Sprintf("%s%d.mkv", dirName, k)
		args := []string{
			"-loop", "1", "-framerate", "2", "-i", ssName, "-i", voiceName, "-c:v", "libx264", "-preset", "medium", "-tune", "stillimage", "-crf", "18", "-c:a", "copy", "-shortest", "-pix_fmt", "yuv420p", spliceName,
		}
		err := exec.Command("ffmpeg", args...).Run()
		if err != nil {
			return errors.Wrap(err, "could not splice files")
		}
	}

	return nil
}
