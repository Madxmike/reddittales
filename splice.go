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
			err = s.cleanup(s.screenshotDir(in))
			if err != nil {
				log.Println(err)
			}
			err = s.cleanup(s.voiceDir(in))
			if err != nil {
				log.Println(err)
			}
			err = s.cleanup(s.outputDir(in))
			if err != nil {
				log.Println(err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Splicer) splice(data Data) error {
	_ = os.Mkdir(s.outputDir(data), os.ModeDir)
	for k := range data.Lines() {
		ssName := fmt.Sprintf("%s%d.png", s.screenshotDir(data), k)
		voiceName := fmt.Sprintf("%s%d.mp3", s.voiceDir(data), k)
		spliceName := fmt.Sprintf("%s%d.mkv", s.outputDir(data), k)
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

func (s *Splicer) cleanup(dir string) error {
	err := os.RemoveAll(dir)
	if err != nil {
		return errors.Wrapf(err, "could not remove all files from %s", dir)
	}
	return nil
}

func (s *Splicer) dir(base string, data Data) string {
	return fmt.Sprintf("%s%s/", base, data.ID)
}

func (s *Splicer) voiceDir(data Data) string {
	return s.dir(s.voiceClipPath, data)
}

func (s *Splicer) screenshotDir(data Data) string {
	return s.dir(s.screenshotPath, data)
}

func (s *Splicer) outputDir(data Data) string {
	return s.dir(s.outputPath, data)
}
