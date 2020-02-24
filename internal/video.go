package internal

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/turnage/graw/reddit"
	stripmd "github.com/writeas/go-strip-markdown"
	"gopkg.in/neurosnap/sentences.v1"
	"gopkg.in/neurosnap/sentences.v1/english"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Generator interface {
	Generate(ctx context.Context) ([]byte, error)
	CreateContext(ctx context.Context) (context.Context, context.CancelFunc)
}

type VideoWorker struct {
	tokenizer *sentences.DefaultSentenceTokenizer
	post      *reddit.Post
	comments  []*reddit.Comment
	clips     []Clip
}

type Clip struct {
	screenshotData []byte
	audioData      []byte
}

func NewVideoWorker(post *reddit.Post, comments []*reddit.Comment) (VideoWorker, error) {
	tokenizer, err := english.NewSentenceTokenizer(nil)
	if err != nil {
		return VideoWorker{}, errors.Wrap(err, "could not create video worker")
	}
	return VideoWorker{
		tokenizer: tokenizer,
		post:      post,
		comments:  comments,
		clips:     make([]Clip, 0),
	}, nil
}

func (vw *VideoWorker) Process(ctx context.Context, screenshotGenerator ScreenshotGenerator, audioGenerator AudioGenerator, finished chan<- []byte) {
	startedTime := time.Now()
	log.Printf("Started video (id: %s) at %s\n", vw.post.ID, startedTime.Format(time.Stamp))
	err := vw.processPost(ctx, screenshotGenerator, audioGenerator)
	if err != nil {
		log.Println(errors.Wrap(err, fmt.Sprintf("could not process post title id: %s", vw.post.ID)))
		return
	}
	log.Printf("Generating clips for %d comments for id: %s\n", len(vw.comments), vw.post.ID)
	var wg sync.WaitGroup
	clipReturn := make(chan []Clip, len(vw.comments))

	for _, c := range vw.comments {
		wg.Add(1)
		go func(comment *reddit.Comment) {
			log.Printf("Generating clips for comment id: %s/%s\n", vw.post.ID, comment.ID)
			clips, err := vw.processComment(ctx, screenshotGenerator, audioGenerator, comment)
			if err != nil {
				log.Println(errors.Wrap(err, fmt.Sprintf("could not process comment id: %s//%s", vw.post.ID, comment.ID)))
			}

			clipReturn <- clips
			wg.Done()
			log.Printf("Finished generating clips for comment id: %s/%s\n", vw.post.ID, comment.ID)

		}(c)
	}

	wg.Wait()
	close(clipReturn)
	log.Printf("Finish generating clips for %s\n", vw.post.ID)
	for clips := range clipReturn {
		vw.clips = append(vw.clips, clips...)
	}
	log.Printf("Generated %d clips for id: %s\n", len(vw.clips), vw.post.ID)

	dirName, err := ioutil.TempDir("", vw.post.ID)
	if err != nil {
		log.Println(errors.Wrap(err, "could not video"))
		return
	}

	defer os.RemoveAll(dirName)
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
	clips, err := vw.processText(ctx, screenshotGenerator, audioGenerator, vw.post.Title)
	if err != nil {
		return errors.Wrap(err, "could not process post title")
	}
	if vw.post.SelfText != "" {
		screenshotGenerator.renderType = SelfPostRender
		clips, err = vw.processText(ctx, screenshotGenerator, audioGenerator, vw.post.SelfText)
		if err != nil {
			return errors.Wrap(err, "could not process post title")
		}
	}
	vw.clips = append(vw.clips, clips...)
	return nil
}

func (vw *VideoWorker) processComment(ctx context.Context, screenshotGenerator ScreenshotGenerator, audioGenerator AudioGenerator, comment *reddit.Comment) ([]Clip, error) {
	screenshotGenerator.renderType = CommentRender
	screenshotGenerator.Karma = comment.Ups
	screenshotGenerator.Username = comment.Author
	clips, err := vw.processText(ctx, screenshotGenerator, audioGenerator, comment.Body)
	if err != nil {
		return nil, errors.Wrap(err, "could not process comment")
	}
	return clips, nil
}

func (vw *VideoWorker) processText(ctx context.Context, screenshotGenerator ScreenshotGenerator, audioGenerator AudioGenerator, text string) ([]Clip, error) {
	clips := make([]Clip, 0)
	text = stripmd.Strip(text)
	text = strings.ReplaceAll(text, "&gt;", "")
	tokens := vw.tokenizer.Tokenize(text)
	screenshotCtx, screenshotCancel := screenshotGenerator.CreateContext(ctx)
	audioCtx, audioCancel := screenshotGenerator.CreateContext(ctx)
	defer screenshotCancel()
	defer audioCancel()
	for _, token := range tokens {
		screenshotGenerator.Text += token.Text
		audioGenerator.Text = token.Text
		clip := Clip{
			screenshotData: make([]byte, 0),
			audioData:      make([]byte, 0),
		}
		//An error here means we should just abandon this clip
		//as it will generate a bad video once stitched
		err := clip.ReadScreenshotData(screenshotCtx, screenshotGenerator)
		if err != nil {
			return nil, errors.Wrap(err, "could not generate clip")
		}
		err = clip.ReadAudioData(audioCtx, audioGenerator)
		if err != nil {
			return nil, errors.Wrap(err, "could not generate clip")
		}
		clips = append(clips, clip)
	}
	return clips, nil
}

func (vw *VideoWorker) StitchClips(dirName string) ([]string, error) {
	//Don't try and throw this into a go-routine. The cpu, memory, and disk will hate
	//you as 300+ ffmpeg instances are launched
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

func (c *Clip) ReadScreenshotData(ctx context.Context, generator Generator) (err error) {
	c.screenshotData, err = generator.Generate(ctx)
	if err != nil {
		return errors.Wrap(err, "could not read screenshot data")
	}
	return nil
}

func (c *Clip) ReadAudioData(ctx context.Context, generator Generator) (err error) {
	c.audioData, err = generator.Generate(ctx)
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
