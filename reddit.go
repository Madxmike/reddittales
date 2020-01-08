package main

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/turnage/graw/reddit"
	stripmd "github.com/writeas/go-strip-markdown"
	"log"
	"strconv"
	"strings"
	"time"
)

type RedditGenerator struct {
	Output     chan Data
	config     redditConfig
	pollTicker *time.Ticker
	reddit     reddit.Bot
}

func NewRedditGenerator(secrets Secrets, config redditConfig, pollInterval time.Duration) (*RedditGenerator, error) {
	grawCfg := reddit.BotConfig{
		Agent: secrets.UserAgent,
		App: reddit.App{
			ID:       secrets.ClientID,
			Secret:   secrets.ClientSecret,
			Username: secrets.Username,
			Password: secrets.Password,
		},
	}
	bot, err := reddit.NewBot(grawCfg)
	if err != nil {
		return nil, errors.Wrap(err, "could not start reddit bot")
	}

	return &RedditGenerator{
		Output:     make(chan Data),
		config:     config,
		pollTicker: time.NewTicker(pollInterval),
		reddit:     bot,
	}, nil
}

func (r *RedditGenerator) Start(ctx context.Context) {
	for {
		select {
		case <-r.pollTicker.C:
			r.poll()
		case <-ctx.Done():
			return
		}
	}
}

func (r *RedditGenerator) poll() {
	for _, subConfig := range r.config.Watched {
		listingParams := map[string]string{
			"limit": strconv.Itoa(subConfig.NumPosts),
			"sort":  subConfig.SortPostsBy,
			"time":  subConfig.MaximumPostTime,
		}
		harvest, err := r.reddit.ListingWithParams(fmt.Sprintf("/r/%s", subConfig.Name), listingParams)
		if err != nil {
			log.Printf("could not retrieve posts for %s: %e", subConfig.Name, err)
			continue
		}
		for _, post := range harvest.Posts {
			err = r.processPost(subConfig, post)
			if err != nil {
				log.Printf("could not process post for %s: %e", subConfig.Name, err)
				continue

			}
		}
	}
}

func (r *RedditGenerator) processPost(subConfig subredditConfig, post *reddit.Post) error {

	post, err := r.reddit.Thread(post.Permalink)
	if err != nil {
		return errors.Wrap(err, "could not get full post thread")
	}

	topLevels := r.filterTopLevel(post.Replies)
	capturedComments := make([]*reddit.Comment, 0, subConfig.NumComments)
	if len(topLevels) < cap(capturedComments) {
		capturedComments = append(capturedComments, topLevels...)
	} else {
		capturedComments = append(capturedComments, topLevels[:cap(capturedComments)]...)
	}

	postData := r.postToData(post)
	postData.Comments = r.commentToData(capturedComments)
	r.Output <- postData
	return nil
}

func (r *RedditGenerator) filterTopLevel(comments []*reddit.Comment) []*reddit.Comment {
	topLevels := make([]*reddit.Comment, 0)
	for _, comment := range comments {
		if comment.IsTopLevel() {
			topLevels = append(topLevels, comment)
		}
	}
	return topLevels
}

func (r *RedditGenerator) sanitizeText(text string) string {
	text = stripmd.Strip(text)
	text = strings.ReplaceAll(text, "&gt;", "")

	return text
}

func (r *RedditGenerator) postToData(post *reddit.Post) Data {
	text := post.SelfText
	if r.config.SanitizeText {
		text = r.sanitizeText(text)
	}
	return Data{
		ID:       post.ID,
		Username: post.Author,
		Score:    int(post.Ups),
		Title:    post.Title,
		Text:     text,
		Comments: make([]Data, 0),
	}
}

func (r *RedditGenerator) commentToData(comments []*reddit.Comment) []Data {
	commentData := make([]Data, 0, len(comments))
	for _, comment := range comments {
		text := comment.Body
		if r.config.SanitizeText {
			text = r.sanitizeText(text)
		}
		commentData = append(commentData, Data{
			ID:       comment.ID,
			Username: comment.Author,
			Score:    int(comment.Ups),
			Title:    "",
			Text:     text,
			Comments: make([]Data, 0),
		})
	}

	return commentData
}
