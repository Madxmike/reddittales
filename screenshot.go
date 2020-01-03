package main

import (
	"fmt"
	sshot "github.com/slotix/pageres-go-wrapper"
	"log"
	"os/exec"
)

func GenerateAllScreenshots(data map[string]Data, render chan<- Data, path string) error {
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
	sshot.GetShots(urls, params)
	for name, d := range data {
		log.Println("generating " + name)

		err := generateScreenshot(name, d, render)
		if err != nil {
			log.Println(err)
		}
	}
	return nil
}

func generateScreenshot(name string, data Data, render chan<- Data) error {
	log.Println("generating " + name)
	render <- data
	exec.Command("gowitness.exe", "single", "--url", "127.0.0.1:3000")

	return nil
}
