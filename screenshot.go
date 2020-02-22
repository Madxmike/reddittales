package main

import (
	"context"
	"fmt"
	"github.com/chromedp/chromedp"
	"github.com/pkg/errors"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

type ScreenshotReader struct {
	client   *http.Client
	Username string
	Karma    int32
	Text     string
}

func (r *ScreenshotReader) takeScreenshot(res *[]byte) chromedp.Tasks {
	URL := url.URL{
		Host: fmt.Sprintf("https://localhost:%s", os.Getenv("PORT")),
	}
	query := URL.Query()
	query.Add("username", r.Username)
	query.Add("karma", strconv.Itoa(int(r.Karma)))
	query.Add("text", r.Text)

	return chromedp.Tasks{
		chromedp.Navigate(URL.String()),
		chromedp.WaitVisible("#main", chromedp.ByID),
		chromedp.Screenshot("#main", res, chromedp.NodeVisible, chromedp.ByID),
	}

}

func (r ScreenshotReader) Read(b []byte) (n int, err error) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()
	err = chromedp.Run(ctx, r.takeScreenshot(&b))
	if err != nil {
		return len(b), errors.Wrap(err, "could not read screenshot")
	}
	return len(b), nil
}
