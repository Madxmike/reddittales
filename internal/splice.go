package internal

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
			err := s.process(in)
			if err != nil {

			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Splicer) process(data Data) error {
	fileNames, err := s.splice(data)
	if err != nil {
		log.Println(err)
	}
	err = s.combine(s.outputDir(data), fileNames)
	err = s.cleanupAll(data)
	if err != nil {
		log.Println(err)
	}
	return nil
}

func (s *Splicer) splice(data Data) ([]string, error) {
	_ = os.Mkdir(s.outputDir(data), os.ModeDir)
	lines := data.Lines()
	fileNames := make([]string, 0)
	for k := range lines {
		outputFileName := fmt.Sprintf("%d.mkv", k)
		ssName := fmt.Sprintf("%s%d.png", s.screenshotDir(data), k)
		voiceName := fmt.Sprintf("%s%d.mp3", s.voiceDir(data), k)
		spliceName := fmt.Sprintf("%s%s", s.outputDir(data), outputFileName)
		err := s.execute("-loop", "1", "-framerate", "2", "-i", ssName, "-i", voiceName, "-c:v", "libx264", "-preset", "medium", "-tune", "stillimage", "-crf", "18", "-c:a", "copy", "-shortest", "-pix_fmt", "yuv420p", spliceName)
		if err != nil {
			return nil, errors.Wrap(err, "could not splice files")
		}

		fileNames = append(fileNames, outputFileName)
	}

	return fileNames, nil
}

func (s *Splicer) combine(path string, fileNames []string) error {
	outputFileName := path + "filenames.txt"
	file, err := os.Create(outputFileName)
	if err != nil {
		return errors.Wrap(err, "could not create filenames file")
	}
	for _, name := range fileNames {

		write := fmt.Sprintf("file '%s' \n", name)
		_, err = file.WriteString(write)
		if err != nil {
			return errors.Wrapf(err, "could not write \"%s\" to file", write)
		}
	}
	file.Close()

	err = s.execute("-f", "concat", "-safe", "0", "-i", outputFileName, "-c", "copy", path+"output.mkv")
	if err != nil {
		return errors.Wrap(err, "could not combine files")
	}
	return nil
}

func (s *Splicer) execute(args ...string) error {
	log.Println("Executing: ffmpeg", args)
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout

	err := cmd.Run()
	if err != nil {
		return errors.Wrap(err, "could not execute ffmpeg")
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

func (s *Splicer) cleanupAll(data Data) error {
	err := s.cleanup(s.screenshotDir(data))
	if err != nil {
		return err
	}
	err = s.cleanup(s.voiceDir(data))
	if err != nil {
		return err
	}
	//err = s.cleanup(s.outputDir(data))
	//if err != nil {
	//	return err
	//}
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
