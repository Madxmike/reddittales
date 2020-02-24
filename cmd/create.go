package cmd

import (
	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"context"
	"github.com/madxmike/reddittales/internal"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"log"
	"net/http"
	"sync"
)

func CreateCmd() *cobra.Command {
	var sort string
	var time string
	var num int
	create := &cobra.Command{
		Use:   "create [subreddit name]",
		Short: "creates a video from the specified parameters",
		Args:  cobra.MinimumNArgs(1),
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
			for _, p := range posts {
				comments, err := rw.GetComments(p, 5)
				if err != nil {
					log.Println(errors.Wrap(err, "could not retrieve post comments"))
					continue
				}
				vw, err := internal.NewVideoWorker(p, comments)
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
			for data := range finished {
				log.Println(len(data))
			}
		},
	}
	create.Flags().StringVar(&sort, "sort", "top", "how to sort subreddit posts")
	create.Flags().StringVar(&time, "time", "day", "filter subreddit posts by time")
	create.Flags().IntVar(&num, "num", 1, "how many posts should be retrieved")

	return create
}
