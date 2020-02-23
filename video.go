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
	"time"
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
	startedTime := time.Now()
	log.Printf("Started video (id: %s) at %s\n", vw.post.ID, startedTime.Format(time.Stamp))
	err := vw.processPost(ctx, screenshotGenerator, audioGenerator)
	if err != nil {
		log.Println(errors.Wrap(err, fmt.Sprintf("could not process post title id: %s", vw.post.ID)))
		return
	}
	for _, c := range vw.comments {
		screenshotGenerator.renderType = CommentRender
		screenshotGenerator.Username = c.Author
		screenshotGenerator.Karma = c.Ups
		err := vw.processText(ctx, screenshotGenerator, audioGenerator, c.Body)
		if err != nil {
			log.Println(errors.Wrap(err, fmt.Sprintf("could not process comment id: %s//%s", vw.post.ID, c.ID)))
			continue
		}
	}

	dirName, err := ioutil.TempDir("", vw.post.ID)
	if err != nil {
		log.Println(errors.Wrap(err, "could not video"))
		return
	}

	//defer os.RemoveAll(dirName)
	stitchedClips, err := vw.StitchClips(dirName)
	if err != nil {
		log.Println(errors.Wrap(err, "could not create video"))
		return
	}
	final, err := vw.finalStitch(stitchedClips, dirName)
	if err != nil {
		log.Println(errors.Wrap(err, "could not create video"))
		return
	}
	log.Printf("Finished video (id: %s) at %s after %s seconds\n", vw.post.ID, time.Now().Format(time.Stamp), time.Now().Sub(startedTime)/time.Second)

	finished <- final
}

func (vw *VideoWorker) processPost(ctx context.Context, screenshotGenerator ScreenshotGenerator, audioGenerator AudioGenerator) error {
	screenshotGenerator.renderType = PostRender
	screenshotGenerator.Karma = vw.post.Ups
	screenshotGenerator.Username = vw.post.Author
	err := vw.processText(ctx, screenshotGenerator, audioGenerator, vw.post.Title)
	if err != nil {
		return errors.Wrap(err, "could not process post title")
	}
	if vw.post.IsSelf {
		screenshotGenerator.renderType = SelfPostRender
		err = vw.processText(ctx, screenshotGenerator, audioGenerator, vw.post.SelfText)
		if err != nil {
			return errors.Wrap(err, "could not process post title")
		}
	}
	return nil
}

func (vw *VideoWorker) processText(ctx context.Context, screenshotGenerator ScreenshotGenerator, audioGenerator AudioGenerator, text string) error {
	splitText := strings.Split(text, "\n")
	for _, line := range splitText {
		screenshotGenerator.Text += line
		audioGenerator.Text = line
		clip := Clip{
			screenshotData: make([]byte, 0),
			audioData:      make([]byte, 0),
		}
		err := clip.Read(ctx, screenshotGenerator, audioGenerator)
		if err != nil {
			//An error here means we should just abandon this clip
			//as it will generate a bad video once stitched
			return errors.Wrap(err, "could not generate clip")
		}
		vw.clips = append(vw.clips, clip)
	}
	return nil
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

func (vw *VideoWorker) finalStitch(stitchedClips []string, dirName string) ([]byte, error) {
	var mux strings.Builder
	for _, clipPath := range stitchedClips {
		_, err := mux.WriteString(fmt.Sprintf("file '%s'\n", clipPath))
		if err != nil {
			return nil, errors.Wrap(err, "could not write mux file")
		}
	}
	muxFilename, err := writeFile(dirName, "*.txt", []byte(mux.String()))
	outputFileName := fmt.Sprintf("%s%coutput.mkv", dirName, os.PathSeparator)
	cmd := exec.Command("ffmpeg",
		"-y",
		"-f", "concat",
		"-safe", "0",
		"-i", muxFilename,
		"-c", "copy",
		outputFileName,
	)
	err = cmd.Run()
	if err != nil {
		return nil, errors.Wrap(err, "could not run stitch command")
	}
	b, err := ioutil.ReadFile(outputFileName)
	if err != nil {
		return nil, errors.Wrap(err, "could load read output file")
	}
	return b, nil
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
	screenshotFileName, err := writeFile(dirPath, "*.png", c.screenshotData)
	if err != nil {
		return "", errors.Wrap(err, "could not create screenshot file")
	}
	audioFileName, err := writeFile(dirPath, "*.mp3", c.audioData)
	if err != nil {
		return "", errors.Wrap(err, "could not audio file")
	}

	outputFileName := fmt.Sprintf("%s%c%s.mkv", dirPath, os.PathSeparator, outputName)
	cmd := exec.Command("ffmpeg",
		"-y",
		"-loop", "1",
		"-framerate", "2",
		"-i", screenshotFileName,
		"-i", audioFileName,
		"-c:v", "libx264",
		"-preset", "medium",
		"-tune", "stillimage",
		"-crf", "18",
		"-c:a", "copy",
		"-shortest",
		"-pix_fmt", "yuv420p",
		"-vf", "scale=1920:1080:force_original_aspect_ratio=decrease,pad=1920:1080:(ow-iw)/2:(oh-ih)/2:#333333",
		outputFileName,
	)
	//cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return "", errors.Wrap(err, "could not run stitch command")
	}

	return outputFileName, nil
}

func writeFile(dirPath, pattern string, b []byte) (string, error) {
	f, err := ioutil.TempFile(dirPath, pattern)
	if err != nil {
		return "", errors.Wrap(err, "could not create file")
	}
	defer f.Close()
	_, err = f.Write(b)
	if err != nil {
		return "", errors.Wrap(err, "could not write data")
	}
	return f.Name(), nil
}
