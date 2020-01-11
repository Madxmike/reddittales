package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/chromedp/chromedp"
	"github.com/pkg/errors"
	"log"
	"net/http"
	"os"
	"sync"
)

type ScreenshotGenerator struct {
	wg         *sync.WaitGroup
	Input      chan Data
	path       string
	serverAddr string
}

func (s *ScreenshotGenerator) Start(ctx context.Context) {
	for {
		select {
		case in := <-s.Input:
			log.Printf("Generating screenshots for %s", in.ID)
			err := s.generateAll(ctx, in, "#post")
			if err != nil {
				log.Println(err)
			}
			log.Printf("Finished generating screenshots for %s", in.ID)
			s.wg.Done()
		case <-ctx.Done():
			return
		}
	}
}

func (s *ScreenshotGenerator) elementScreenshot(urlstr, sel string, res *[]byte) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.EmulateViewport(3840, 4320),
		chromedp.Navigate(urlstr),
		chromedp.ScrollIntoView("#post", chromedp.NodeVisible, chromedp.ByQuery),
		chromedp.Screenshot(sel, res, chromedp.NodeVisible, chromedp.ByQuery),
	}
}

func (s *ScreenshotGenerator) generateAll(ctx context.Context, data Data, selector string) error {

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
		err := s.generate(ctx, serverData, selector, fmt.Sprintf("%s/%d.png", dirName, n))
		if err != nil {
			return errors.Wrap(err, "could not generate screenshot")
		}
	}

	for _, comment := range data.Comments {
		comment.ID = fmt.Sprintf("%s/%s", data.ID, comment.ID)
		err := s.generateAll(ctx, comment, selector)
		if err != nil {
			return errors.Wrap(err, "could not generate comment")
		}
	}

	return nil
}

func (s *ScreenshotGenerator) generate(ctx context.Context, data Data, selector string, filename string) error {
	//TODO - attach to a chromium headless image instead
	//chromeCtx, cancel := chromedp.NewContext(ctx)
	//defer cancel()
	err := s.sendData(data)
	if err != nil {
		return errors.Wrap(err, "could not generate screenshot")
	}
	//var b []byte
	//err = chromedp.Run(chromeCtx, s.elementScreenshot(s.serverAddr, selector, &b))
	//if err != nil {
	//	return errors.Wrap(err, "could not take screenshot")
	//}
	//
	//err = ioutil.WriteFile(filename, b, 0777)
	//if err != nil {
	//	return errors.Wrap(err, "could not save screenshot")
	//}

	return nil
}

func (s *ScreenshotGenerator) sendData(data Data) error {
	d, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "could not marshal data")
	}
	_, err = http.Post(s.serverAddr+"/push", "application/json", bytes.NewBuffer(d))
	if err != nil {
		return errors.Wrap(err, "could not post data")
	}
	return nil
}
