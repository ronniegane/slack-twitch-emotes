package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
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
		fmt.Printf("Unable to fetch Twitch emotes list from %s\n", twitchEmotesURL)
		log.Fatal(err)
	}
	twitchBody, err := ioutil.ReadAll(twitchResp.Body)
	if err != nil {
		fmt.Printf("Error reading response body from %s\n", twitchEmotesURL)
		log.Fatal(err)
	}

	// Twitch emotes get returned as a map where the emote names are the keys and also the "name" value
	twitchEmotes := map[string]twitchEmote{}
	err = json.Unmarshal(twitchBody, &twitchEmotes)
	if err != nil {
		fmt.Printf("Error unmarshalling JSON from Twitch")
		log.Fatal(err)
	}

	// Transform this into a simple list of emote IDs and names
	emotes := []twitchEmote{}
	for _, v := range twitchEmotes {
		emotes = append(emotes, v)
	}
	// fmt.Println(emotes)
	fmt.Printf("There are %d Twitch emotes to upload\n", len(emotes))

	// Gather BTTV emotes if requested
	// BTTV emotes are in a structure closer to our desired list of emotes
	bttvEmotes := struct {
		URLTemplate string      `json:"urlTemplate"` // has emote ID and {{image}} (size eg 1x)
		Emotes      []bttvEmote `json:"emotes"`
	}{}
	if !skipBttv {
		bttvEmotesURL := "https://api.betterttv.net/2/emotes"
		bttvResp, err := http.Get(bttvEmotesURL)
		if err != nil {
			log.Fatalf("Unable to fetch BTTV emotes list from %s\n", bttvEmotesURL)
		}
		bttvBody, err := ioutil.ReadAll(bttvResp.Body)
		if err != nil {
			fmt.Printf("Error reading response body from %s\n", bttvEmotesURL)
			log.Fatal(err)
		}

		err = json.Unmarshal(bttvBody, &bttvEmotes)
		if err != nil {
			fmt.Printf("Error unmarshalling JSON from BTTV")
			log.Fatal(err)
		}
		// fmt.Println(bttvEmotes.Emotes)
		fmt.Printf("There are %d BTTV emotes to upload\n", len(bttvEmotes.Emotes))
	}

	// Upload emotes to Slack workspace
	// Fetch a session token for the API
	// To do this we have to fmt into the customization page of our workspace

	/*
		when logging in through the web browser
		seems to POST to root team domain with signin = 1, password, email in JSON
		responds with a bunch of cookies
	*/

	teamURL := "https://" + team + ".slack.com"
	http.Get(teamURL + "/customize/emoji")

	/*
	   If you successfully sign in to the Slack web page, then the /customize/emoji page content
	   has an "api_token" field embedded in one of the scripts.
	   We can successfully make an upload to /api/emoji.add if we have that token in the form data
	*/

	// hardcode token for now
	token := ""
	// We can't just give slack a URL to fetch images from, we have to download the file ourselves and then upload it to Slack
	client := http.DefaultClient

	// Twitch emotes are at https://static-cdn.jtvnw.net/emoticons/v1/{id}/1.0
	for i, e := range emotes {
		// just upload one image while testing
		if i > 0 {
			break
		}
		twitchFetchURL := fmt.Sprintf("https://static-cdn.jtvnw.net/emoticons/v1/%d/1.0", e.ID)
		fmt.Println("Fetching from " + twitchFetchURL)
		resp, err := http.Get(twitchFetchURL)
		if err != nil {
			fmt.Println("Error fetching image from " + twitchFetchURL)
		} else {
			image, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println("Error reading response body from " + twitchFetchURL)
			} else {
				upload(client, image, e.Code, teamURL, token)
			}

		}

	}

	// BTTV emotes are found using the template URL which looks like "//cdn.betterttv.net/emote/{{id}}/{{image}}"
	bttvEmotes.URLTemplate = "https:" + strings.Replace(strings.Replace(bttvEmotes.URLTemplate, "{{id}}", "%s", 1), "{{image}}", "1x", 1)
	if !skipBttv {
		for i, e := range bttvEmotes.Emotes {
			// just upload one image while testing
			if i > 0 {
				break
			}
			BTTVfetchURL := fmt.Sprintf(bttvEmotes.URLTemplate, e.ID)
			fmt.Println("Fetching from " + BTTVfetchURL)
			resp, err := http.Get(BTTVfetchURL)
			if err != nil {
				fmt.Println("Error fetching image from " + BTTVfetchURL)
			} else {
				image, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					fmt.Println("Error reading response body from " + BTTVfetchURL)
				} else {
					upload(client, image, e.Code, teamURL, token)
				}
			}
		}
	}
}

func upload(client *http.Client, image []byte, name, teamURL, token string) {
	// Using MultiPart Writer to make a multipart form data POST request
	var buf bytes.Buffer
	mpw := multipart.NewWriter(&buf)
	w, err := mpw.CreateFormFile("image", name+".png")
	_, err = w.Write(image)
	if err != nil {
		fmt.Println("Unable to load image data for " + name)
		return
	}
	mpw.WriteField("mode", "data")
	mpw.WriteField("name", strings.ToLower(name))
	mpw.WriteField("token", token)
	mpw.Close()
	req, _ := http.NewRequest("POST", teamURL+"/api/emoji.add", &buf)
	req.Header.Set("Content-Type", mpw.FormDataContentType())

	// viewing dump of request
	dump, _ := httputil.DumpRequestOut(req, true)
	fmt.Println(string(dump))

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}

	// Slack returns a 200 with "ok" and "error" fields if there is something wrong
	// so maybe we should get that field and print it
	var data map[string]interface{}
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &data)
	if e, ok := data["error"]; ok {
		fmt.Printf("error: %s", e)
	}

	// Viewing dump of response
	fmt.Println()
	respDump, _ := httputil.DumpResponse(resp, true)
	dumpStr := string(respDump)
	fmt.Println(dumpStr)
}
