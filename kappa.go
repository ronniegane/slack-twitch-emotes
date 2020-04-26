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

	"gopkg.in/yaml.v3"
)

// BTTV emote IDs are hexadecimal (so strings in JSON) while Twitch emote IDs are ints
type bttvEmote struct {
	ID   string `json:"id"`
	Code string `json:"code"`
}

type emojiPack struct {
	Title  string  `yaml:"title"`
	Emojis []emoji `yaml:"emojis"`
}

type emoji struct {
	Name string `yaml:"name"`
	Src  string `yaml:"src"`
}

func main() {
	var team, token string
	flag.StringVar(&team, "team", "", "your team or workspace name")
	flag.StringVar(&token, "token", "", "the user access token from the configuration page")
	flag.Parse()

	// Team and token are required
	if len(team) == 0 || len(token) == 0 {
		fmt.Println("Team name and access token are required")
		os.Exit(1)
	}

	// Read emotes from file
	yamlFile, err := ioutil.ReadFile("test.yaml")

	var emojis emojiPack
	if err != nil {
		fmt.Printf("Unable to read YAML file")
		log.Fatal(err)
	}

	err = yaml.Unmarshal(yamlFile, &emojis)

	// Upload emotes to Slack workspace
	teamURL := "https://" + team + ".slack.com"

	/*
	   If you successfully sign in to the Slack web page, then the /customize/emoji page content
	   has an "api_token" field embedded in one of the scripts.
	   We can successfully make an upload to /api/emoji.add if we have that token in the form data
	*/

	// We can't just give slack a URL to fetch images from, we have to download the file ourselves and then upload it to Slack
	client := http.DefaultClient

	for _, e := range emojis.Emojis {
		fmt.Println("Fetching from " + e.Src)
		resp, err := http.Get(e.Src)
		if err != nil {
			fmt.Println("Error fetching image from " + e.Src)
		} else {
			image, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println("Error reading response body from " + e.Src)
			} else {
				upload(client, image, e.Name, teamURL, token)
			}
		}
	}

}

func bttv(client *http.Client, teamURL string, token string) {
	// Fetch and upload the BTTV emotes
	// BTTV emotes are in a structure closer to our desired list of emotes
	bttvEmotes := struct {
		URLTemplate string      `json:"urlTemplate"` // has emote ID and {{image}} (size eg 1x)
		Emotes      []bttvEmote `json:"emotes"`
	}{}

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

	// BTTV emotes are found using the template URL which looks like "//cdn.betterttv.net/emote/{{id}}/{{image}}"
	bttvEmotes.URLTemplate = "https:" + strings.Replace(strings.Replace(bttvEmotes.URLTemplate, "{{id}}", "%s", 1), "{{image}}", "1x", 1)

	for _, e := range bttvEmotes.Emotes {
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
}
