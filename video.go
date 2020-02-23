package main

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/turnage/graw/reddit"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type Generator interface {
	Generate(ctx context.Context) ([]byte, error)
}

type VideoWorker struct {
	post     *reddit.Post
	comments []*reddit.Comment
	clips    []Clip
}

type Clip struct {
	screenshotData []byte
	audioData      []byte
}

func newVideoWorker(post *reddit.Post, comments []*reddit.Comment) VideoWorker {
	return VideoWorker{
		post:     post,
		comments: comments,
		clips:    make([]Clip, 0),
	}
}

func (vw *VideoWorker) Process(ctx context.Context, screenshotGenerator ScreenshotGenerator, audioGenerator AudioGenerator, finished chan<- []byte) {
Comment:
	for _, c := range vw.comments {
		screenshotGenerator.renderType = CommentRender
		screenshotGenerator.Username = c.Author
		screenshotGenerator.Karma = c.Ups

		//TODO - Implement an actual processing lib here to split text naturally
		splitText := strings.Split(c.Body, "\n")
		for _, line := range splitText {
			screenshotGenerator.Text += line
			audioGenerator.Text += line
			clip := Clip{
				screenshotData: make([]byte, 0),
				audioData:      make([]byte, 0),
			}
			err := clip.Read(ctx, screenshotGenerator, audioGenerator)
			if err != nil {
				//An error here means we should just abandon this comment
				//as it will generate a bad video once stitched
				log.Println(errors.Wrap(err, "could not generate clip"))
				continue Comment
			}
			vw.clips = append(vw.clips, clip)
		}
	}

	dirName, err := ioutil.TempDir("", vw.post.ID)
	if err != nil {
		log.Println(errors.Wrap(err, "could not generate clip"))
		return
	}
	log.Println(dirName)

	defer os.RemoveAll(dirName)
	stitchedClips, err := vw.StitchClips(dirName)
	if err != nil {
		log.Println(errors.Wrap(err, "could not generate clip"))
		return
	}
	final, err := vw.finalStitch(stitchedClips)
	if err != nil {
		log.Println(errors.Wrap(err, "could not generate clip"))
		return
	}

	finished <- final
}

func (vw *VideoWorker) StitchClips(dirName string) ([]string, error) {
	stitchedClips := make([]string, 0, len(vw.clips))
	for k, clip := range vw.clips {
		stitched, err := clip.Stitch(dirName, strconv.Itoa(k))
		if err != nil {
			return nil, errors.Wrap(err, "could not stitch clips")
		}
		stitchedClips = append(stitchedClips, stitched)
	}
	return stitchedClips, nil
}

func (vw *VideoWorker) finalStitch(stitchedClips []string) ([]byte, error) {
	log.Println(stitchedClips)
	final := make([]byte, 0)
	//TODO - Call FFMPEG to stitch clips together
	return final, nil
}

func (c *Clip) Read(ctx context.Context, screenshotGen Generator, audioGen Generator) (err error) {
	c.screenshotData, err = screenshotGen.Generate(ctx)
	if err != nil {
		return errors.Wrap(err, "could not read screenshot data")
	}
	c.audioData, err = audioGen.Generate(ctx)
	if err != nil {
		return errors.Wrap(err, "could not read audio data")
	}
	return nil
}

//Call ffmpeg and stitch the audio and image data into one video
func (c *Clip) Stitch(dirPath, outputName string) (string, error) {
	screenshotFileName, err := c.writeFile(dirPath, "*.png", c.screenshotData)
	if err != nil {
		return "", errors.Wrap(err, "could not create screenshot file")
	}
	audioFileName, err := c.writeFile(dirPath, "*.mp3", c.audioData)
	if err != nil {
		return "", errors.Wrap(err, "could not audio file")
	}

	outputFileName := fmt.Sprintf("%s%c%s.mkv", dirPath, os.PathSeparator, outputName)
	cmd := exec.Command("ffmpeg",
		"-r", "1",
		"-loop", "1",
		"-i", screenshotFileName,
		"-i", audioFileName,
		"-acodec", "copy",
		"-r", "1",
		"-shortest",
		"-vf", "scale=1920:1080",
		outputFileName,
	)
	//cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return "", errors.Wrap(err, "could not run stitch command")
	}

	return outputFileName, nil
}

func (c *Clip) writeFile(dirPath, pattern string, b []byte) (string, error) {
	f, err := ioutil.TempFile(dirPath, pattern)
	if err != nil {
		return "", errors.Wrap(err, "could not create file")
	}
	defer f.Close()
	_, err = f.Write(b)
	if err != nil {
		return "", errors.Wrap(err, "could not write data")
	}
	log.Println(f.Name())
	return f.Name(), nil
}
