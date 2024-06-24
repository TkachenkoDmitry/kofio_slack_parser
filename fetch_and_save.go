package main

import (
	"coffee_order/parser"
	"coffee_order/slack_client"
	"encoding/csv"
	"log"
	"os"
)

type linkStruct struct {
	Link, Src, Price, Title string
}

func fetchAndPersist(config Config, channelId, timestamp string) error {
	slackClient := slack_client.NewSlackClient(slack_client.Config{
		SlackToken:       config.SlackToken,
		SlackCookieValue: config.SlackCookieValue,
	})
	links, err := slackClient.GetOrderUrls(channelId, timestamp)
	if err != nil {
		log.Printf("Error: %s\n", err.Error())
		return err
	}

	linksMap := make(map[string]linkStruct)

	for _, link := range links {
		if _, ok := linksMap[link.Link]; ok {
			continue
		}
		log.Printf("Adding link: %s\n", link.Link)
		linksMap[link.Link] = linkStruct{}
	}

	kofioCredentials := parser.Credentials{
		Username: config.KofioUserName,
		Password: config.KofioPassword,
	}
	parser := parser.NewParser(kofioCredentials)
	err = parser.Init()
	if err != nil {
		log.Printf("Error: %s\n", err.Error())
		return err
	}

	fetchedImages := make(chan linkStruct)
	for link := range linksMap {
		go linkFetcher(link, parser, fetchedImages)
	}

	csvData := [][]string{
		{"User", "Title", "Link", "Price", "Image"},
	}

	for range len(linksMap) {
		resp := <-fetchedImages

		linksMap[resp.Link] = resp
	}

	for _, link := range links {
		csvData = append(
			csvData,
			[]string{link.User, linksMap[link.Link].Title, link.Link, linksMap[link.Link].Price, linksMap[link.Link].Src},
		)
	}

	err = saveCsvData(csvData)
	if err != nil {
		log.Printf("Error saving csv data: %s\n", err.Error())
		return err
	}

	return nil
}

func linkFetcher(link string, parser *parser.Parser, respChan chan linkStruct) {
	imgSrc, err := parser.GetImage(link)
	if err != nil {
		log.Printf("Error: %s\n", err.Error())
	}
	respChan <- linkStruct{Link: link, Src: imgSrc.Image, Price: imgSrc.Price, Title: imgSrc.Title}
}

func saveCsvData(csvData [][]string) error {
	file, err := os.Create("./orders.csv")
	if err != nil {
		log.Printf("Error creating file: %s\n", err.Error())
	}
	defer file.Close()
	csvWriter := csv.NewWriter(file)

	err = csvWriter.WriteAll(csvData)
	if err != nil {
		return err
	}

	log.Printf("Orders saved to orders.csv\n")

	return nil
}
