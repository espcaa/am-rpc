package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/joho/godotenv"
	itunessearch "github.com/mattn/itunes-search-api"
	discordrpc "github.com/rikkuness/discord-rpc"
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

var musicLengthScript = `
tell application "Music"
	set currentTrack to the current track
	set trackLength to the duration of currentTrack
	return trackLength
end tell
`

var albumNameScript = `
tell application "Music"
	set currentTrack to the current track
	set albumName to the album of currentTrack
	return albumName
end tell
`

var currentPositionScript = `
tell application "Music"
	set currentTrack to the current track
	set trackPosition to the player position
	return trackPosition
end tell
`

type CacheItem struct {
	ArtistName string
	TrackName  string
}

var cache CacheItem

func main() {
	godotenv.Load()
	var discordID = os.Getenv("DISCORD_APP_ID")
	client, err := discordrpc.New(discordID)
	if err != nil {
		panic(err)
	}
	var artworkurl, artisturl, imageurl string
	var trackDuration time.Duration

	for {
		trackOutput, err1 := executeAppleScript(trackScript)
		if err1 != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		artistOutput, err2 := executeAppleScript(artistScript)
		if err2 != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		albumOutput, err3 := executeAppleScript(albumNameScript)
		if err3 != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		albumOutput = cleanOutput(albumOutput)
		trackOutput = cleanOutput(trackOutput)
		artistOutput = cleanOutput(artistOutput)

		if cache.ArtistName != artistOutput || cache.TrackName != trackOutput {
			results, err := itunessearch.Search(artistOutput+" "+trackOutput+" "+albumOutput, "FR", "music")
			if err == nil && len(results.Results) > 0 {
				artworkurl = results.Results[0].ArtworkUrl100
				artisturl = results.Results[0].ArtistViewUrl
				imageurl, _ = getArtistImage(artisturl)
			}

			lengthOutput, err := executeAppleScript(musicLengthScript)
			trackDuration = 0
			if err == nil {
				dur, parseErr := time.ParseDuration(cleanOutput(lengthOutput) + "s")
				if parseErr == nil {
					trackDuration = dur
				}
			}

			client.SetActivity(discordrpc.Activity{
				Details: trackOutput,
				State:   artistOutput,
				Assets: &discordrpc.Assets{
					LargeImage: artworkurl,
					LargeText:  artistOutput,
					SmallImage: imageurl,
					SmallText:  trackOutput,
				},
				Timestamps: &discordrpc.Timestamps{
					Start: &discordrpc.Epoch{Time: time.Now()},
					End:   &discordrpc.Epoch{Time: time.Now().Add(trackDuration)},
				},
				Type: 2,
			})

			cache.ArtistName = artistOutput
			cache.TrackName = trackOutput
		} else {
			posOutput, err := executeAppleScript(currentPositionScript)
			if err == nil {
				posSec, parseErr := time.ParseDuration(cleanOutput(posOutput) + "s")
				if parseErr == nil {
					client.SetActivity(discordrpc.Activity{
						Details: trackOutput,
						State:   artistOutput,
						Assets: &discordrpc.Assets{
							LargeImage: artworkurl,
							SmallImage: imageurl,
							SmallText:  trackOutput,
						},
						Timestamps: &discordrpc.Timestamps{
							Start: &discordrpc.Epoch{Time: time.Now().Add(-posSec)},
							End:   &discordrpc.Epoch{Time: time.Now().Add(trackDuration - posSec)},
						},
						Type: 2,
					})
				}
			}
		}

		time.Sleep(1 * time.Second)
	}
}

func executeAppleScript(script string) (string, error) {
	out, err := exec.Command("osascript", "-e", script).Output()
	return string(out), err
}

func cleanOutput(output string) string {
	if len(output) == 0 {
		return ""
	}
	return output[:len(output)-1]
}

func getArtistImage(url string) (string, error) {
	res, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}
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
