package main

import (
	"context"
	"github.com/pkg/errors"
	"github.com/turnage/graw/reddit"
	"log"
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
	voiceData      []byte
}

func newVideoWorker(post *reddit.Post, comments []*reddit.Comment) VideoWorker {
	return VideoWorker{
		post:     post,
		comments: comments,
		clips:    make([]Clip, 0),
	}
}

func (vw *VideoWorker) Process(ctx context.Context, screenshotGenerator ScreenshotGenerator, audioGenerator AudioGenerator, finished chan<- []byte) {
	clips := make([]Clip, 0)
Comment:
	for _, c := range vw.comments {
		screenshotGenerator.renderType = CommentRender
		screenshotGenerator.Username = c.Author
		screenshotGenerator.Karma = c.Ups

		//TODO - Implement an actual processing lib here to split text naturally
		splitText := strings.Split(c.Body, "\n")
		log.Println(len(splitText))
		for _, line := range splitText {
			screenshotGenerator.Text += line
			audioGenerator.Text += line
			clip := Clip{
				screenshotData: make([]byte, 0),
				voiceData:      make([]byte, 0),
			}
			err := clip.Read(ctx, screenshotGenerator, audioGenerator)
			if err != nil {
				//An error here means we should just abandon this comment
				//as it will generate a bad video once stitched
				log.Println(errors.Wrap(err, "could not generate clip"))
				continue Comment
			}
			clips = append(clips, clip)
		}
	}

	stitchedClips, err := vw.StitchClips()
	if err != nil {
		log.Println(errors.Wrap(err, "could not generate clip"))
		return
	}
	final, err := vw.finalStitch(stitchedClips)
	if err != nil {
		log.Println(errors.Wrap(err, "could not generate clip"))
	}
	finished <- final
}

func (vw *VideoWorker) StitchClips() ([][]byte, error) {
	stitchedClips := make([][]byte, 0, len(vw.clips))
	for _, clip := range vw.clips {
		stitched, err := clip.Stitch()
		if err != nil {
			return nil, errors.Wrap(err, "could not stitch clips")
		}
		stitchedClips = append(stitchedClips, stitched)
	}
	return stitchedClips, nil
}

func (vw *VideoWorker) finalStitch(stitchedClips [][]byte) ([]byte, error) {
	final := make([]byte, 0)
	//TODO - Call FFMPEG to stitch clips together
	return final, nil
}

func (c *Clip) Read(ctx context.Context, screenshotGen Generator, audioGen Generator) (err error) {
	c.screenshotData, err = screenshotGen.Generate(ctx)
	if err != nil {
		return errors.Wrap(err, "could not read screenshot data")
	}
	c.voiceData, err = audioGen.Generate(ctx)
	if err != nil {
		return errors.Wrap(err, "could not read audio data")
	}
	return nil
}

//Call ffmpeg and stitch the audio and image data into one video
func (c *Clip) Stitch() ([]byte, error) {
	stitched := make([]byte, 0)
	//TODO - Call ffmpeg to stitch clip together

	return stitched, nil
}
