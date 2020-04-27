# Twitch Emotes for Slack
Is your slack channel boring and lifeless? Having trouble getting your real emotions across with the standard emojis?
Now you can upload all current Twitch and BTTV emotes to your Slack workspace so you can LUL at the good times, pepehands at the bad times, and kappa all the time.

Largely inspired by the Ruby app [caldrealabs/kappa-slack](https://github.com/calderalabs/kappa-slack).

Use of the Slack API inspired by [smashwilson/slack-mojinator](https://github.com/smashwilson/slack-emojinator) and [jackellenberger/emojme](https://github.com/jackellenberger/emojme).

Uses the undocumented BetterTTV API for BTTV emotes.

## Installation
Requires the yaml library: `go get gopkg.in/yaml.v3`

## File format
A list of emotes to upload are expected in the [lambtron/emojipacks](https://github.com/lambtron/emojipacks) style:
```yaml
title: Twitch
emojis:
  - name: babyrage
    src: https://static-cdn.jtvnw.net/emoticons/v1/22639/3.0
  - name: biblethump
    src: https://static-cdn.jtvnw.net/emoticons/v1/86/3.0
```

## Usage
1. Fetch your token:
    1. open your workspace's customisation page in a browser
    2. open up JS console
    3. run `window.prompt("your api token is: ", TS.boot_data.api_token)` and copy the value (should look like `xoxs...`)
2. Run with `go run kappa.go` with the following flags:
    - `team` : the name of your team, eg "abc" if your slack workspace is "abc.slack.com"
    - `token` : the token from step 1
    - `file` : a YAML file containing emotes to upload, formatted as above
