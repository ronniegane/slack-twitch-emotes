package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

// BTTV emote IDs are hexadecimal (so strings in JSON) while Twitch emote IDs are ints
type bttvEmote struct {
	ID   string `json:"id"`
	Code string `json:"code"`
}
type twitchEmote struct {
	ID   int    `json:"id"`
	Code string `json:"code"`
}

func main() {
	var team, email, password string
	var skipBttv bool
	flag.StringVar(&team, "team", "", "your team or workspace name")
	flag.StringVar(&email, "email", "", "the email address you use for this slack team")
	flag.StringVar(&password, "password", "", "your password for this slack team")
	flag.BoolVar(&skipBttv, "skip-bttv", false, "skip BTTV emotes and only get Twitch ones")
	flag.Parse()

	// Team and email address are required
	if len(team) == 0 || len(email) == 0 {
		fmt.Println("Team name and email address are required")
		os.Exit(1)
	}

	// If password is missing then ask for it
	for len(password) == 0 {
		fmt.Printf("Password for %s in %s: ", email, team)
		bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			fmt.Printf("Failed to read password: %v", err)
		}
		password = string(bytePassword)
		fmt.Println()
	}

	fmt.Printf("team: %q, email: %q, password: %q\n", team, email, password)

	// Gather Twitch emotes
	twitchEmotesURL := "https://twitchemotes.com/api_cache/v3/global.json"
	twitchResp, err := http.Get(twitchEmotesURL)
	if err != nil {
		log.Printf("Unable to fetch Twitch emotes list from %s\n", twitchEmotesURL)
		log.Fatal(err)
	}
	twitchBody, err := ioutil.ReadAll(twitchResp.Body)
	if err != nil {
		log.Printf("Error reading response body from %s\n", twitchEmotesURL)
		log.Fatal(err)
	}

	// Twitch emotes get returned as a map where the emote names are the keys and also the "name" value
	twitchEmotes := map[string]twitchEmote{}
	err = json.Unmarshal(twitchBody, &twitchEmotes)
	if err != nil {
		log.Printf("Error unmarshalling JSON from Twitch")
		log.Fatal(err)
	}

	// Transform this into a simple list of emote IDs and names
	emotes := []twitchEmote{}
	for _, v := range twitchEmotes {
		emotes = append(emotes, v)
	}
	fmt.Println(emotes)

	// Gather BTTV emotes if requested
	if !skipBttv {
		bttvEmotesURL := "https://api.betterttv.net/2/emotes"
		bttvResp, err := http.Get(bttvEmotesURL)
		if err != nil {
			log.Fatalf("Unable to fetch BTTV emotes list from %s\n", bttvEmotesURL)
		}
		bttvBody, err := ioutil.ReadAll(bttvResp.Body)
		if err != nil {
			log.Printf("Error reading response body from %s\n", bttvEmotesURL)
			log.Fatal(err)
		}
		// BTTV emotes are in a structure closer to our desired list of emotes
		bttvEmotes := struct {
			URLTemplate string      `json:"urlTemplate"`
			Emotes      []bttvEmote `json:"emotes"`
		}{}

		err = json.Unmarshal(bttvBody, &bttvEmotes)
		if err != nil {
			log.Printf("Error unmarshalling JSON from BTTV")
			log.Fatal(err)
		}
		fmt.Println(bttvEmotes.Emotes)
	}

	// Upload emotes to Slack workspace
}
