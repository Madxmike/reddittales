package main

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/neurosnap/sentences.v1/english"
	"io/ioutil"
	"os"
	"strings"
)

type Data struct {
	ID       string `json:"id,omitempty"`
	Username string `json:"username"`
	Score    int    `json:"score"`
	Title    string `json:"title"`
	Text     string `json:"text"`
	Comments []Data `json:"comments"`
}

func (d Data) Sentences() []string {
	tokenizer, err := english.NewSentenceTokenizer(nil)
	if err != nil {
		//TODO - handle this error
		panic(err)
	}

	tokens := tokenizer.Tokenize(d.Text)
	//Sentences -> strings here for easy use in the rest of the program
	sentences := make([]string, 0, len(tokens))
	for i := range tokens {
		sentences = append(sentences, tokens[i].Text)
	}
	return sentences
}

func LoadAllData(path string) ([]Data, error) {
	fileInfos, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, errors.Wrap(err, "could not load data dir")
	}

	allData := make([]Data, 0)
	for _, info := range fileInfos {
		if info.IsDir() {
			continue
		}
		data, err := loadData(path + info.Name())
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("could not load data for file (%s)", info.Name()))
		}

		if data.ID == "" {
			data.ID = strings.TrimSuffix(info.Name(), ".json")
		}
		allData = append(allData, data)
	}

	return allData, nil
}

func loadData(fileName string) (data Data, err error) {
	file, err := os.Open(fileName)
	if err != nil {
		return
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(&data)
	if err != nil {
		return
	}

	return
}
