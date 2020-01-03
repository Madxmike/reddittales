package main

import (
	"fmt"
	"log"
	"os/exec"
)

func SplitAll(data map[string]Data) error {
	log.Println("Generating Spliced files")
	for name, d := range data {
		_ = Splice(name, d, PATH_VOICE_CLIPS, PATH_SCREEN_SHOTS, "spliced/")
	}
	log.Println("Finished Generating Spliced Files")
	return nil
}

func Combine(name string, data Data, splicePath string) error {

	return nil
}

func Splice(name string, data Data, voicePath string, screenshotPath string, splicePath string) error {
	for k := range SplitText(data.Text) {
		fileName := fmt.Sprintf("%s_%d", name, k)
		ssName := fmt.Sprintf("%s%s.png", screenshotPath, fileName)
		voiceName := fmt.Sprintf("%s%s.mp3", voicePath, fileName)
		spliceName := fmt.Sprintf("%s%s.mkv", splicePath, fileName)
		args := []string{
			"-loop",
			"1",
			"-framerate",
			"2",
			"-i",
			ssName,
			"-i",
			voiceName,
			"-c:v",
			"libx264",
			"-preset",
			"medium",
			"-tune",
			"stillimage",
			"-crf",
			"18",
			"-c:a",
			"copy",
			"-shortest",
			"-pix_fmt",
			"yuv420p",
			spliceName,
		}
		err := exec.Command("ffmpeg", args...).Run()
		if err != nil {
			log.Println(err)
		}
	}

	return nil
}
