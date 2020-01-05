package main

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
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
				log.Println(err)
				continue
			}
			for k, comment := range in.Comments {
				comment.ID = fmt.Sprintf("%s/%d", in.ID, k)
				err = s.process(comment)
				if err != nil {
					log.Println(err)
					continue
				}
			}
			err = s.spliceTitle(in)
			if err != nil {
				log.Println(err)
				continue
			}
			err = s.combineFinal(in)
			if err != nil {
				log.Println(err)
				continue
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Splicer) process(data Data) error {
	s.createDir(data)
	fileNames, err := s.splice(data)
	if err != nil {
		log.Println(err)
	}
	err = s.combine(s.outputDir(data), fileNames)
	if err != nil {
		log.Println(err)
	}

	//err = s.cleanupAll(data)

	return nil
}
func (s *Splicer) createDir(data Data) {
	_ = os.Mkdir(s.outputPath+data.ID, os.ModeDir)
}

func (s *Splicer) splice(data Data) ([]string, error) {
	lines := data.Lines()
	fileNames := make([]string, 0)
	for k := range lines {
		outputFileName := fmt.Sprintf("%d.mkv", k)
		ssName := fmt.Sprintf("%s%d.png", s.screenshotDir(data), k)
		voiceName := fmt.Sprintf("%s%d.mp3", s.voiceDir(data), k)
		spliceName := fmt.Sprintf("%s%s", s.outputDir(data), outputFileName)
		err := s.executeSplit(ssName, voiceName, spliceName)
		if err != nil {
			return nil, errors.Wrap(err, "failed to splice")
		}
		fileNames = append(fileNames, outputFileName)
	}
	return fileNames, nil
}

func (s *Splicer) spliceTitle(data Data) error {
	if data.Title == "" {
		return nil
	}
	title := "title"
	outputFileName := "output.mkv"
	ssName := fmt.Sprintf("%s%s.png", s.screenshotDir(data), title)
	voiceName := fmt.Sprintf("%s%s.mp3", s.voiceDir(data), title)
	spliceName := fmt.Sprintf("%s%s", s.outputDir(data), outputFileName)

	err := s.executeSplit(ssName, voiceName, spliceName)
	if err != nil {
		return errors.Wrap(err, "failed to splice")
	}

	return nil
}

func (s *Splicer) executeSplit(screenshotFileName string, voiceClipFileName string, splicedFileName string) error {
	err := s.execute("-loop", "1", "-framerate", "2", "-i", screenshotFileName, "-i", voiceClipFileName, "-c:v", "libx264", "-preset", "medium", "-tune", "stillimage", "-crf", "18", "-c:a", "copy", "-shortest", "-pix_fmt", "yuv420p", splicedFileName)
	if err != nil {
		return errors.Wrap(err, "could not splice files")
	}
	return nil
}

func (s *Splicer) combine(path string, fileNames []string) error {
	outputFileName, err := s.writeFileNames(path, fileNames)
	if err != nil {
		return errors.Wrap(err, "could not combine files")
	}
	err = s.execute("-f", "concat", "-safe", "0", "-i", outputFileName, "-c", "copy", path+"output.mkv")
	if err != nil {
		return errors.Wrap(err, "could not combine files")
	}
	return nil
}

func (s *Splicer) combineFinal(data Data) error {
	outfilesName := "output.mkv"
	basePath := fmt.Sprintf("%s%s/", s.outputPath, data.ID)
	fileNames := []string{outfilesName}
	for k := range data.Comments {
		fileNames = append(fileNames, fmt.Sprintf("%d/%s", k, outfilesName))
	}
	outputFileName, err := s.writeFileNames(basePath, fileNames)
	if err != nil {
		return errors.Wrap(err, "could not combine files")
	}

	err = s.execute("-f", "concat", "-safe", "0", "-i", outputFileName, "-c", "copy", basePath+"finished.mkv")
	if err != nil {
		return errors.Wrap(err, "could not combine files")
	}
	return nil
}

func (s *Splicer) writeFileNames(path string, fileNames []string) (string, error) {
	if len(fileNames) == 0 {
		return "", nil
	}
	outputFileName := path + "filenames.txt"
	b := make([]byte, 0)
	for _, name := range fileNames {
		write := fmt.Sprintf("file '%s'\n", name)
		b = append(b, []byte(write)...)
	}
	err := ioutil.WriteFile(outputFileName, b, 0777)
	if err != nil {
		return "", errors.Wrap(err, "could not write file names")
	}
	return outputFileName, nil
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
