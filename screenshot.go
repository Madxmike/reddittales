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

type RenderType string

const (
	PostRender     RenderType = "post"
	SelfPostRender            = "self_post"
	CommentRender             = "comment"
)

type ScreenshotGenerator struct {
	client     *http.Client
	renderType RenderType
	Username   string
	Karma      int32
	Text       string
}

func (r *ScreenshotGenerator) takeScreenshot(res *[]byte) chromedp.Tasks {
	URL := url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("localhost:%s", os.Getenv("PORT")),
		Path:   "/screenshot",
	}
	query := URL.Query()
	query.Add("render", string(r.renderType))
	query.Add("username", r.Username)
	query.Add("karma", strconv.Itoa(int(r.Karma)))
	query.Add("text", r.Text)

	URL.RawQuery = query.Encode()
	return chromedp.Tasks{
		chromedp.Navigate(URL.String()),
		chromedp.WaitVisible("#main", chromedp.ByID),
		chromedp.Screenshot("#main", res, chromedp.NodeVisible, chromedp.ByID),
	}

}

func (r ScreenshotGenerator) Generate(ctx context.Context) ([]byte, error) {
	ctx, cancel := chromedp.NewContext(ctx)
	defer cancel()
	var b []byte
	err := chromedp.Run(ctx, r.takeScreenshot(&b))
	if err != nil {
		return nil, errors.Wrap(err, "could not generate screenshot")
	}
	return b, nil
}
