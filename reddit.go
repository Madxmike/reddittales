package main

import (
	"context"
	"github.com/jzelinskie/geddit"
	"github.com/pkg/errors"
	stripmd "github.com/writeas/go-strip-markdown"
	"log"
	"strings"
	"time"
)

type RedditGenerator struct {
	Output     chan Data
	config     redditConfig
	reddit     *geddit.OAuthSession
	pollTicker *time.Ticker
}

func NewRedditGenerator(secrets Secrets, config redditConfig, pollInterval time.Duration) (*RedditGenerator, error) {
	OAuth, err := geddit.NewOAuthSession(secrets.ClientID, secrets.ClientSecret, secrets.UserAgent, "")
	if err != nil {
		return nil, errors.Wrap(err, "could not establish reddit connection")
	}
	err = OAuth.LoginAuth(secrets.Username, secrets.Password)
	if err != nil {
		return nil, errors.Wrap(err, "could not login to reddit")
	}

	return &RedditGenerator{
		Output:     make(chan Data),
		config:     config,
		reddit:     OAuth,
		pollTicker: time.NewTicker(pollInterval),
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
		options := geddit.ListingOptions{
			Time:  geddit.ThisDay,
			Limit: subConfig.NumPosts,
			Count: subConfig.NumPosts,
		}
		posts, err := r.reddit.SubredditSubmissions(subConfig.Name, geddit.TopSubmissions, options)
		if err != nil {
			log.Println(errors.Wrapf(err, "could not get posts for %s", subConfig.Name))
			continue
		}
		for _, post := range posts {
			err = r.processSubmission(subConfig, post)
			if err != nil {
				log.Println(errors.Wrapf(err, "could not process post %s", post.ID))
			}
		}
	}
}

func (r *RedditGenerator) processSubmission(subConfig subredditConfig, submission *geddit.Submission) error {
	log.Printf("Processing submission %s", submission.ID)
	submissionData := r.submissionToData(submission)

	options := geddit.ListingOptions{
		Limit: subConfig.NumComments,
		Count: subConfig.NumComments,
	}
	commentData := make([]Data, 0, subConfig.NumComments)

	for len(commentData) < subConfig.NumComments {
		log.Println("Retrieving Comments")
		comments, err := r.reddit.Comments(submission, geddit.PopularitySort(sub.Comments.Sort), options)
		if err != nil {
			return errors.Wrap(err, "could not retrieve comments")
		}
		//We are out of comments
		if len(comments) == 0 {
			break
		}
		lastComment := comments[len(comments)-1]
		options.After = lastComment.FullID
		remaining := subConfig.NumComments - len(commentData)
		options.Limit = remaining
		options.Count = remaining
		commentData = append(commentData, r.commentToData(comments)...)
		log.Printf("%d / %d comments collected", len(commentData), subConfig.NumComments)
	}

	log.Printf("%d comments collected", len(commentData))

	if r.config.SanitizeText {
		submissionData.Text = r.sanitizeText(submissionData.Text)
		for _, c := range commentData {
			c.Text = r.sanitizeText(c.Text)
		}
	}
	submissionData.Comments = commentData
	r.Output <- submissionData
	return nil
}

func (r *RedditGenerator) sanitizeText(text string) string {
	text = stripmd.Strip(text)
	text = strings.ReplaceAll(text, "&gt;", "")

	return text
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
