package main

import (
	"fmt"
	sshot "github.com/slotix/pageres-go-wrapper"
	"log"
)

func GenerateAllScreenshots(data map[string]Data, server *Server, path string) error {
	log.Println("Generating Screenshots")
	params := sshot.Parameters{
		Command:   "pageres",
		Sizes:     "860x1000",
		Crop:      "--crop",
		Scale:     "--scale 1",
		Timeout:   "--timeout 30",
		UserAgent: "",
	}
	urls := []string{
		"http://127.0.0.1:3000",
	}
	for name, d := range data {
		err := generateScreenshot(name, d, server, path, params, urls)
		if err != nil {
			log.Println(err)
		}
	}

	log.Println("Finished Generating Shootshots")
	return nil
}

func generateScreenshot(name string, data Data, server *Server, path string, params sshot.Parameters, urls []string) error {
	log.Printf("Generating Screenshot for %s\n", data.Title)
	d := data
	d.Text = ""
	splitText := SplitText(data.Text)
	//baseHeight := 200
	//height := baseHeight + (len(splitText)
	for k, text := range splitText {

		d.Text += text
		server.data = d
		params.Filename = fmt.Sprintf("--filename=%s/%s_%d", path, name, k)

		sshot.GetShots(urls, params)
	}
	return nil
}
