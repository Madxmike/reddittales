package main

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/turnage/graw/reddit"
	"strconv"
	"time"
)

type RedditWorker struct {
	reddit.Bot
}

func NewRedditWorker(agentFile string) (*RedditWorker, error) {
	conn, err := reddit.NewBotFromAgentFile(agentFile, 5*time.Second)
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to reddit")
	}
	return &RedditWorker{Bot: conn}, nil
}

func (r *RedditWorker) ScrapePosts(subreddit string, sortBy string, age string, numPosts int) ([]*reddit.Post, error) {
	listingParams := map[string]string{
		"limit": strconv.Itoa(numPosts),
		"sort":  sortBy,
		"time":  age,
	}
	subreddit = fmt.Sprintf("/r/%s", subreddit)
	harvest, err := r.ListingWithParams(subreddit, listingParams)
	if err != nil {
		return nil, errors.Wrap(err, "could not scrape posts")
	}

	return harvest.Posts, nil
}

func (r *RedditWorker) GetComments(post *reddit.Post, amount int, filters ...func(c *reddit.Comment) bool) ([]*reddit.Comment, error) {
	post, err := r.Thread(post.Permalink)
	if err != nil {
		return nil, errors.Wrap(err, "could not get full post")
	}

	comments := make([]*reddit.Comment, 0, amount)
	for _, reply := range post.Replies {
		if len(comments) < cap(comments) {
			for _, filter := range filters {
				if !filter(reply) {
					continue
				}
				comments = append(comments, reply)
			}
		}
	}
	return comments, nil
}

func FilterDistinguished(c *reddit.Comment) bool {
	return c.Distinguished != ""
}

func FilterKarma(karma int32) func(c *reddit.Comment) bool {
	return func(c *reddit.Comment) bool {
		return c.Ups >= karma
	}
}
