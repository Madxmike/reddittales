package cmd

import (
	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"context"
	"fmt"
	"github.com/madxmike/reddittales/internal"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/turnage/graw/reddit"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

func CreateCmd() *cobra.Command {
	var sort string
	var time string
	var num int
	var outputPath string
	var outputFiletype string
	var maximumNumComments int
	var filterDistiguishedComments bool
	var commentKarmaThreshold int
	var backgroundFile string
	var intermissionFile string
	var musicFile string
	create := &cobra.Command{
		Use:   "create [subreddit name]",
		Short: "creates a video from the specified parameters",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			rw, err := internal.NewRedditWorker(agentFile)
			if err != nil {
				QuitError(errors.Wrap(err, "could not create reddit worker"))
			}
			posts, err := rw.ScrapePosts(args[0], sort, time, num)
			if err != nil {
				QuitError(errors.Wrap(err, "could not scrape reddit posts"))
			}
			go internal.StartServer(port)

			finished := make(chan []byte, 0)
			var wg sync.WaitGroup
			ctx := context.Background()
			ttsClient, err := texttospeech.NewClient(ctx)
			if err != nil {
				panic(errors.Wrap(err, "could not create tts client"))
			}
			screenshotGenerator := internal.ScreenshotGenerator{
				Client: http.DefaultClient,
			}
			audioGenerator := internal.AudioGenerator{
				Client: ttsClient,
			}
			filters := []func(comment *reddit.Comment) bool{
				internal.FilterKarma(int32(commentKarmaThreshold)),
			}
			if filterDistiguishedComments {
				filters = append(filters, internal.FilterDistinguished)
			}

			for _, p := range posts {
				comments, err := rw.GetComments(p, maximumNumComments, filters...)
				if err != nil {
					log.Println(errors.Wrap(err, "could not retrieve post comments"))
					continue
				}
				vw, err := internal.NewVideoWorker(p, comments, backgroundFile, intermissionFile, musicFile)
				if err != nil {
					log.Println(errors.Wrap(err, "could not process video"))
					continue
				}
				wg.Add(1)
				go func() {
					vw.Process(ctx, screenshotGenerator, audioGenerator, finished)
					wg.Done()
				}()
			}

			go func() {
				wg.Wait()
				close(finished)
			}()
			fileNum := 0
			for data := range finished {
				fileName := fmt.Sprintf("output_%d.%s", fileNum, outputFiletype)
				if outputPath != "" {
					outputPath = strings.TrimSuffix(outputPath, string(os.PathSeparator))
					fileName = fmt.Sprintf("%s%c%s", outputPath, os.PathSeparator, fileName)
				}
				err = ioutil.WriteFile(fileName, data, os.ModePerm)
				if err != nil {
					log.Println(errors.Wrap(err, "could not write output"))
					continue
				}
			}
		},
	}
	create.Flags().StringVar(&sort, "sort", "top", "how to sort subreddit posts")
	create.Flags().StringVar(&time, "time", "day", "filter subreddit posts by time")
	create.Flags().IntVar(&num, "num", 1, "how many posts should be retrieved")
	create.Flags().StringVar(&outputPath, "outputPath", "", "the output directory of created files")
	create.Flags().StringVar(&outputFiletype, "filetype", "mkv", "the file type of the finished file")
	create.Flags().IntVar(&maximumNumComments, "maximumNumComments", 10, "the maximum number of comments included")
	create.Flags().BoolVar(&filterDistiguishedComments, "filterDistinguished", true, "filter distinguished comments i.e. automod")
	create.Flags().IntVar(&commentKarmaThreshold, "commentKarmaThreshold", 1000, "the minimum karma required for a comment to be included")

	create.Flags().StringVar(&backgroundFile, "backgroundFile", "", "the path to the background image or video file (NYI)")
	create.Flags().StringVar(&intermissionFile, "intermissionFile", "", "the path to the intermission video file (NYI)")
	create.Flags().StringVar(&musicFile, "musicFile", "", "the path to the music file (NYI)")
	return create
}
