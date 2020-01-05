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
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Splicer) process(data Data) error {
	dirName := fmt.Sprintf("%s%s/", s.outputPath, data.ID)
	_ = os.Mkdir(dirName, os.ModeDir)
	for _, comment := range data.Comments {
		comment.ID = fmt.Sprintf("%s/%s", data.ID, comment.ID)
		err := s.process(comment)
		if err != nil {
			return errors.Wrap(err, "could not process comment")
		}
	}

	outputFilename, err := s.stitchAV(data)
	if err != nil {
		return errors.Wrap(err, "could not process data")
	}

	log.Println(outputFilename)

	return nil
}

func (s *Splicer) stitchAV(data Data) (string, error) {
	var outputFilename string
	lines := data.Lines()
	if data.Title != "" {
		lines = append([]string{data.Title}, lines...)
	}

	for k := range lines {
		outputFilename = fmt.Sprintf("%d.mkv", k)
		screenshotFilename := fmt.Sprintf("%s%d.png", s.screenshotDir(data), k)
		voiceclipFilename := fmt.Sprintf("%s%d.mp3", s.voiceDir(data), k)
		stitchedFilename := fmt.Sprintf("%s%s", s.outputDir(data), outputFilename)
		err := s.executeStitch(screenshotFilename, voiceclipFilename, stitchedFilename)
		if err != nil {
			return outputFilename, errors.Wrap(err, "could not execute stitch")
		}
	}
	return outputFilename, nil
}

func (s *Splicer) executeStitch(screenshotFilename, voiceClipFilename, stitchFilename string) error {
	err := s.execute("-loop", "1", "-framerate", "2", "-i", screenshotFilename, "-i", voiceClipFilename, "-c:v", "libx264", "-preset", "medium", "-tune", "stillimage", "-crf", "18", "-c:a", "copy", "-shortest", "-pix_fmt", "yuv420p", "-vf", "scale=1920:-2", stitchFilename)
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
