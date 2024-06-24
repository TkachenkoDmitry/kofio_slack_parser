package main

import (
	"flag"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	SlackToken, SlackCookieValue, KofioUserName, KofioPassword string
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	slackToken := os.Getenv("SLACK_TOKEN")
	if slackToken == "" {
		log.Fatalf("Error: SLACK_TOKEN is not set\n")
	}
	slackCookieValue := os.Getenv("SLACK_COOKIE_VALUE")
	if slackCookieValue == "" {
		log.Fatalf("Error: SLACK_COOKIE_VALUE is not set\n")
	}
	kofioUserName := os.Getenv("KOFIO_USER_NAME")
	if kofioUserName == "" {
		log.Fatalf("Error: KOFIO_USER_NAME is not set\n")
	}
	kofioPassword := os.Getenv("KOFIO_PASSWORD")
	if kofioPassword == "" {
		log.Fatalf("Error: KOFIO_PASSWORD is not set\n")
	}

	config := Config{
		SlackToken:       slackToken,
		SlackCookieValue: slackCookieValue,
		KofioUserName:    kofioUserName,
		KofioPassword:    kofioPassword,
	}
	threadUrl := flag.String("thread-url", "", "Slack thread URL")
	flag.Parse()
	if threadUrl == nil || *threadUrl == "" {
		filename := filepath.Base(os.Args[0])

		log.Printf("Usage: %s -thread-url <URL>\n", filename)
		return
	}

	url, err := url.Parse(*threadUrl)
	if err != nil {
		log.Printf("Error: %s\n", err.Error())
	}

	parts := strings.Split(url.Path, "/")

	channelId := parts[2]
	rawTimestamp := parts[3]

	rawTimestamp = strings.TrimPrefix(rawTimestamp, "p")

	timestamp := ""
	for i, s := range strings.Split(rawTimestamp, "") {
		if i == 10 {
			timestamp += "."
		}
		timestamp += s
	}

	err = fetchAndPersist(config, channelId, timestamp)
	if err != nil {
		log.Fatalf("Error: %s\n", err.Error())
	}
}
