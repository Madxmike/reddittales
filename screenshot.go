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
		Sizes:     "1920x130",
		Crop:      "--crop",
		Scale:     "--scale 0.9",
		Timeout:   "--timeout 30",
		Filename:  fmt.Sprintf("--filename=%s/<%%= url %%>", path),
		UserAgent: "",
	}
	urls := []string{
		"http://127.0.0.1:3000",
	}
	for name, d := range data {
		err := generateScreenshot(name, d, server, params, urls)
		if err != nil {
			log.Println(err)
		}
	}

	log.Println("Finished Generating Shootshots")
	return nil
}

func generateScreenshot(name string, data Data, server *Server, params sshot.Parameters, urls []string) error {
	log.Printf("Generating Screenshot for %s\n", data.Title)
	d := data
	d.Text = ""
	for _, text := range SplitText(data.Text) {
		d.Text += text
		log.Println(d.Text)
		server.data = d
		sshot.GetShots(urls, params)
	}
	return nil
}
