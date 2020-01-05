package main

import (
	"encoding/json"
	"github.com/pkg/errors"
	"io/ioutil"
)

type Config struct {
	Server     server      `json:"server"`
	Subreddits []subreddit `json:"subreddits"`
}

type server struct {
	Port string `json:"port"`
}

type comments struct {
	Count                int    `json:"count"`
	Sort                 string `json:"sort"`
	IncludeTopSubComment string `json:"includeTopSubComment"`
}

type subreddit struct {
	Name     string   `json:"name"`
	Count    int      `json:"count"`
	Comments comments `json:"comments"`
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
