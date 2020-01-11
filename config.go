package main

import (
	"encoding/json"
	"github.com/pkg/errors"
	"io/ioutil"
)

type Config struct {
	Server serverConfig `json:"server"`
	Reddit redditConfig `json:"reddit"`
	Voice  voiceConfig  `json:"voice"`
	Stitch stitchConfig `json:"stitch"`
}

type serverConfig struct {
	Port            string `json:"port"`
	TemplatePath    string `json:templatePath`
	RefreshTemplate bool   `json:"refreshTemplate"`
}

type redditConfig struct {
	PollDelay    int               `json:"pollDelayMinutes"`
	SanitizeText bool              `json:"sanitizeText"`
	Watched      []subredditConfig `json:"watched"`
}

type subredditConfig struct {
	Name            string `json:"name"`
	NumPosts        int    `json:"numPosts"`
	SortPostsBy     string `json:"sortPostsBy"`
	MaximumPostTime string `json:"maximumPostTime"`
	NumComments     int    `json:"numComments"`
	SortCommentsBy  string `json:"sortCommentsBy"`
}

type voiceConfig struct {
}

type stitchConfig struct {
	FinishedFilePath string `json:"finishedFilePath"`
}

type Secrets struct {
	UserAgent    string `json:"user_agent"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Username     string `json:"username"`
	Password     string `json:"password"`
}

func LoadSecrets(fileName string) (Secrets, error) {
	var secrets Secrets
	b, err := ioutil.ReadFile(fileName)
	if err != nil {
		return secrets, errors.Wrap(err, "could not load secrets")
	}
	err = json.Unmarshal(b, &secrets)
	if err != nil {
		return secrets, errors.Wrap(err, "could not unmarshal secrets")
	}
	return secrets, nil
}

func LoadConfig(fileName string) (Config, error) {
	var config Config
	b, err := ioutil.ReadFile(fileName)
	if err != nil {
		return config, errors.Wrap(err, "could not load config")
	}
	err = json.Unmarshal(b, &config)
	if err != nil {
		return config, errors.Wrap(err, "could not unmarshal config")
	}
	return config, nil
}
