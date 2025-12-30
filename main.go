package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/hugolgst/rich-go/client"
	"github.com/joho/godotenv"
	itunessearch "github.com/mattn/itunes-search-api"
)

var trackScript = `
tell application "Music"
	set currentTrack to the current track
	set trackName to the name of currentTrack
	return trackName
end tell
`

var artistScript = `
tell application "Music"
	set currentTrack to the current track
	set artistName to the artist of currentTrack
	return artistName
end tell
`

func main() {

	godotenv.Load()

	// connect to discord
	var discordID = os.Getenv("DISCORD_APP_ID")
	err := client.Login(discordID)
	if err != nil {
		panic(err)
	}
	// infinite loop :3
	for {
		println("updating song")
		// execute with osascript
		trackOutput, err1 := executeAppleScript(trackScript)
		if err1 != nil {
			println("Error getting track:", err.Error())
			continue
		}

		artistOutput, err2 := executeAppleScript(artistScript)
		if err2 != nil {
			println("Error getting artist:", err.Error())
			continue
		}
		trackOutput = cleanOutput(trackOutput)
		artistOutput = cleanOutput(artistOutput)

		// get album cover url

		results, err3 := itunessearch.Search(artistOutput+" "+trackOutput, "US", "music")
		if err3 != nil || len(results.Results) == 0 {
			println("Error getting album cover:", err.Error())
			continue
		}

		var artworkurl string
		var artisturl string
		if len(results.Results) != 0 {
			artworkurl = results.Results[0].ArtworkUrl100
			artisturl = results.Results[0].ArtistViewUrl
		}

		// scrape the artist page to get picture url

		var imageurl, err = getArtistImage(artisturl)
		if err != nil {
			println("Error getting artist image:", err.Error())
		}

		// print output

		println("Current Track:", trackOutput)
		println("Current Artist:", artistOutput)
		println(
			"-------------------------",
		)

		// send to discord

		err = client.SetActivity(client.Activity{
			Details:    artistOutput + " - " + trackOutput,
			LargeImage: artworkurl,
			LargeText:  trackOutput,
			SmallImage: imageurl,
		})

		// sleep for 5 seconds
		time.Sleep(1 * time.Second)
	}
}

func executeAppleScript(script string) (string, error) {
	execCmd := exec.Command("osascript", "-e", script)
	output, err := execCmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// clean output func
func cleanOutput(output string) string {
	if len(output) == 0 {
		return ""
	}
	return output[:len(output)-1]
}

func getArtistImage(url string) (string, error) {
	// fetch page
	res, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return "", fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// parse HTML
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return "", err
	}

	image, exists := doc.Find(`meta[property="og:image"]`).Attr("content")
	if !exists {
		return "", fmt.Errorf("artist image not found")
	}

	return image, nil
}
