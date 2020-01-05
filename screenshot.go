package main

import (
	"context"
	"fmt"
	"github.com/chromedp/chromedp"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"os"
	"sync"
)

type ScreenshotGenerator struct {
	wg         *sync.WaitGroup
	Input      chan Data
	path       string
	serverAddr string
	serverSend chan<- Data
}

func (s *ScreenshotGenerator) Start(ctx context.Context) {
	ctx, _ = chromedp.NewContext(ctx)
	for {
		select {
		case in := <-s.Input:
			ctx, cancel := chromedp.NewContext(context.Background())
			err := s.generate(ctx, in, "#post")
			if err != nil {
				log.Println(err)
			}
			s.wg.Done()
			cancel()
		case <-ctx.Done():
			return
		}
	}
}

func (s *ScreenshotGenerator) elementScreenshot(urlstr, sel string, res *[]byte) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.EmulateViewport(1920, 1080),
		chromedp.Navigate(urlstr),
		chromedp.WaitVisible(sel, chromedp.ByID),
		chromedp.Screenshot(sel, res, chromedp.NodeVisible, chromedp.ByID),
	}
}

func (s *ScreenshotGenerator) generate(ctx context.Context, data Data, selector string) error {
	log.Println(data.ID)
	dirName := fmt.Sprintf("%s%s/", s.path, data.ID)
	_ = os.Mkdir(dirName, os.ModeDir)

	lines := data.Lines()
	if data.Title != "" {
		lines = append([]string{data.Title}, lines...)
	}
	serverData := data
	serverData.Text = ""
	for n, text := range lines {
		serverData.Text += text
		var b []byte
		s.serverSend <- serverData
		err := chromedp.Run(ctx, s.elementScreenshot(s.serverAddr, selector, &b))
		if err != nil {
			return errors.Wrap(err, "could not take screenshot")
		}

		filename := fmt.Sprintf("%s/%d.png", dirName, n)
		err = ioutil.WriteFile(filename, b, 0777)
		if err != nil {
			return errors.Wrap(err, "could not save screenshot")
		}

	}

	for _, comment := range data.Comments {
		comment.ID = fmt.Sprintf("%s/%s", data.ID, comment.ID)
		err := s.generate(ctx, comment, selector)
		if err != nil {
			return errors.Wrap(err, "could not generate comment")
		}
	}

	return nil
}
