package internal

import (
	"context"
	"github.com/jzelinskie/geddit"
	"github.com/pkg/errors"
	"log"
	"time"
)

type Secrets struct {
	UserAgent    string `json:"user_agent"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Username     string `json:"username"`
	Password     string `json:"password"`
}

type RedditGenerator struct {
	reddit            *geddit.OAuthSession
	Output            chan Data
	pollTicker        *time.Ticker
	sort              geddit.PopularitySort
	listingOptions    geddit.ListingOptions
	watchedSubreddits []string
}

func NewRedditGenerator(secrets Secrets, pollInterval time.Duration, sort geddit.PopularitySort, options geddit.ListingOptions, watchedSubreddit ...string) (*RedditGenerator, error) {
	OAuth, err := geddit.NewOAuthSession(secrets.ClientID, secrets.ClientSecret, secrets.UserAgent, "")
	if err != nil {
		return nil, errors.Wrap(err, "could not establish reddit connection")
	}
	err = OAuth.LoginAuth(secrets.Username, secrets.Password)
	if err != nil {
		return nil, errors.Wrap(err, "could not login to reddit")
	}

	return &RedditGenerator{
		reddit:            OAuth,
		Output:            make(chan Data),
		pollTicker:        time.NewTicker(pollInterval),
		sort:              sort,
		listingOptions:    options,
		watchedSubreddits: watchedSubreddit,
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
	for _, sub := range r.watchedSubreddits {
		posts, err := r.reddit.SubredditSubmissions(sub, r.sort, r.listingOptions)
		if err != nil {
			log.Println(errors.Wrapf(err, "could not get posts for %s", sub))
			continue
		}
		for _, post := range posts {
			data := r.toData(post)
			r.Output <- data
		}
	}
}

func (r *RedditGenerator) toData(post *geddit.Submission) Data {
	return Data{
		ID:       post.ID,
		Username: post.Author,
		Score:    post.Score,
		Title:    post.Title,
		Text:     post.Selftext,
	}
}
