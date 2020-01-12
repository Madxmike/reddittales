package main

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

type Splicer struct {
	Config stitchConfig
	Input  chan Data
}

func (s *Splicer) Start(ctx context.Context) {
	for {
		select {
		case in := <-s.Input:
			processedFilenames, err := s.process(in)
			if err != nil {
				log.Println(err)
				continue
			}

			path := fmt.Sprintf("%s%c%s%c", os.TempDir(), os.PathSeparator, in.ID, os.PathSeparator)
			for i := range processedFilenames {
				processedFilenames[i] = strings.TrimPrefix(processedFilenames[i], path)
			}
			finalFilename, err := s.stitchVideo(path, s.reverseNames(processedFilenames), in.ID)
			if err != nil {
				log.Println(err)
				continue
			}
			err = s.moveFinishedFile(path, finalFilename)
			if err != nil {
				log.Println(err)
				continue
			}
			err = s.cleanupAll(in)
			if err != nil {
				log.Println(err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Splicer) process(data Data) ([]string, error) {
	processedFilenames := make([]string, 0)
	dirName := fmt.Sprintf("%s%c%s%c", os.TempDir(), os.PathSeparator, data.ID, os.PathSeparator)
	_ = os.Mkdir(dirName, os.ModeDir)
	for _, comment := range data.Comments {
		comment.ID = fmt.Sprintf("%s%c%s", data.ID, os.PathSeparator, comment.ID)
		processed, err := s.process(comment)
		if err != nil {
			return nil, errors.Wrap(err, "could not process comment")
		}
		processedFilenames = append(processedFilenames, processed...)
	}

	outputFilenames, err := s.stitchAV(data)
	if err != nil {
		return nil, errors.Wrap(err, "could not process data")
	}

	processed, err := s.stitchVideo(dirName, outputFilenames, "output")
	if err != nil {
		return nil, errors.Wrap(err, "could not stitch video")
	}

	return append(processedFilenames, processed), nil
}

func (s *Splicer) stitchVideo(dirName string, filenames []string, outputFilename string) (string, error) {
	var processedFilename string
	b := make([]byte, 0)
	for _, name := range filenames {
		write := []byte(fmt.Sprintf("file '%s%s'\n", dirName, name))
		b = append(b, write...)
	}
	muxFilename := fmt.Sprintf("%s%s", dirName, "filenames.txt")
	err := ioutil.WriteFile(muxFilename, b, 0777)
	if err != nil {
		return processedFilename, errors.Wrap(err, "could not write filenames file")
	}
	processedFilename = fmt.Sprintf("%s%s.mp4", dirName, outputFilename)
	err = s.execute("-y", "-f", "concat", "-safe", "0", "-i", muxFilename, "-c", "copy", processedFilename)
	if err != nil {
		return processedFilename, errors.Wrap(err, "could not combine files")
	}

	return processedFilename, nil
}

func (s *Splicer) stitchAV(data Data) ([]string, error) {
	sentences := data.Sentences()
	outputFilenames := make([]string, 0, len(sentences))

	if data.Title != "" {
		sentences = append([]string{data.Title}, sentences...)
	}

	for k := range sentences {
		outputFilename := fmt.Sprintf("%d.mp4", k)
		screenshotFilename := fmt.Sprintf("%s%d.png", s.screenshotDir(data), k)
		voiceclipFilename := fmt.Sprintf("%s%d.mp3", s.voiceDir(data), k)
		stitchedFilename := fmt.Sprintf("%s%s", s.outputDir(data), outputFilename)
		err := s.executeStitch(screenshotFilename, voiceclipFilename, stitchedFilename)
		if err != nil {
			return nil, errors.Wrap(err, "could not execute stitch")
		}
		outputFilenames = append(outputFilenames, outputFilename)
	}
	return outputFilenames, nil
}

func (s *Splicer) moveFinishedFile(path string, filename string) error {
	err := os.Rename(filename, s.Config.FinishedFilePath+strings.TrimPrefix(filename, path))
	if err != nil {
		return errors.Wrap(err, "could not move file")
	}
	return nil
}

func (s *Splicer) executeStitch(screenshotFilename, voiceClipFilename, stitchFilename string) error {
	err := s.execute("-y", "-loop", "1", "-framerate", "2", "-i", screenshotFilename, "-i", voiceClipFilename, "-c:v", "libx264", "-preset", "medium", "-tune", "stillimage", "-crf", "18", "-c:a", "copy", "-shortest", "-pix_fmt", "yuv420p", "-vf", "scale=1920:1080:force_original_aspect_ratio=decrease,pad=1920:1080:(ow-iw)/2:(oh-ih)/2:#333333", stitchFilename)
	if err != nil {
		return errors.Wrap(err, "could not splice files")
	}
	return nil
}

func (s *Splicer) execute(args ...string) error {
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout

	err := cmd.Run()
	if err != nil {
		log.Println(args)
		return errors.Wrap(err, "could not execute ffmpeg")
	}

	return nil
}

func (s *Splicer) reverseNames(filenames []string) []string {
	for i, j := 0, len(filenames)-1; i < j; i, j = i+1, j-1 {
		filenames[i], filenames[j] = filenames[j], filenames[i]
	}

	return filenames
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
	err = s.cleanup(s.outputDir(data))
	if err != nil {
		return err
	}
	return nil
}

func (s *Splicer) dir(base string, data Data) string {
	return fmt.Sprintf("%s%c%s%c", base, os.PathSeparator, data.ID, os.PathSeparator)
}

func (s *Splicer) voiceDir(data Data) string {
	return s.dir(os.TempDir(), data)
}

func (s *Splicer) screenshotDir(data Data) string {
	return s.dir(os.TempDir(), data)
}

func (s *Splicer) outputDir(data Data) string {
	return s.dir(os.TempDir(), data)
}
