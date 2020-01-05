package main

import (
	"context"
	"github.com/jzelinskie/geddit"
	"github.com/pkg/errors"
	"log"
	"time"
)

type RedditGenerator struct {
	reddit     *geddit.OAuthSession
	Output     chan Data
	pollTicker *time.Ticker
	subreddits []subreddit
}

func NewRedditGenerator(secrets Secrets, pollInterval time.Duration, subreddits []subreddit) (*RedditGenerator, error) {
	OAuth, err := geddit.NewOAuthSession(secrets.ClientID, secrets.ClientSecret, secrets.UserAgent, "")
	if err != nil {
		return nil, errors.Wrap(err, "could not establish reddit connection")
	}
	err = OAuth.LoginAuth(secrets.Username, secrets.Password)
	if err != nil {
		return nil, errors.Wrap(err, "could not login to reddit")
	}

	return &RedditGenerator{
		reddit:     OAuth,
		Output:     make(chan Data),
		pollTicker: time.NewTicker(pollInterval),
		subreddits: subreddits,
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
	for _, sub := range r.subreddits {
		options := geddit.ListingOptions{
			Time:  geddit.ThisDay,
			Limit: sub.Count,
			Count: sub.Count,
		}
		posts, err := r.reddit.SubredditSubmissions(sub.Name, geddit.TopSubmissions, options)
		if err != nil {
			log.Println(errors.Wrapf(err, "could not get posts for %s", sub.Name))
			continue
		}
		for _, post := range posts {
			err = r.processSubmission(sub, post)
			if err != nil {
				log.Println(errors.Wrapf(err, "could not process post %s", post.ID))
			}
		}
	}
}

func (r *RedditGenerator) processSubmission(sub subreddit, submission *geddit.Submission) error {
	submissionData := r.submissionToData(submission)

	if sub.Comments.Count > 0 {
		options := geddit.ListingOptions{
			Count: sub.Comments.Count,
			Limit: sub.Comments.Count,
		}

		comments, err := r.reddit.Comments(submission, geddit.PopularitySort(sub.Comments.Sort), options)
		if err != nil {
			return errors.Wrap(err, "could not retrieve comments")
		}
		submissionData.Comments = r.commentToData(comments)
	}

	r.Output <- submissionData
	return nil
}

func (r *RedditGenerator) submissionToData(submission *geddit.Submission) Data {
	return Data{
		ID:       submission.ID,
		Username: submission.Author,
		Score:    submission.Score,
		Title:    submission.Title,
		Text:     submission.Selftext,
		Comments: make([]Data, 0),
	}
}

func (r *RedditGenerator) commentToData(comments []*geddit.Comment) []Data {
	commentData := make([]Data, 0, len(comments))
	for _, comment := range comments {
		commentData = append(commentData, Data{
			ID:       comment.FullID,
			Username: comment.Author,
			Score:    int(comment.Score),
			Title:    "",
			Text:     comment.Body,
			Comments: make([]Data, 0),
		})
	}

	return commentData
}
