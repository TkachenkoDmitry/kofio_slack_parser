package slack_client

import (
	"errors"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/slack-go/slack"
)

type LinkWithUser struct {
	User string
	Link string
}

type SlackClient struct {
	accessToken string
	cookieValue string
}

func (c *SlackClient) getHttpClient() (*http.Client, error) {
	cookieJar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	cookie := http.Cookie{
		Name:   "d",
		Value:  c.cookieValue,
		Path:   "/",
		Domain: "slack.com",
	}

	cookies := []*http.Cookie{&cookie}
	cookies = append(cookies, &cookie)

	url := url.URL{
		Scheme: "https",
		Host:   "slack.com",
	}
	cookieJar.SetCookies(&url, cookies)

	httpClient := &http.Client{
		Jar:     cookieJar,
		Timeout: 0,
	}

	return httpClient, nil
}

func getRichtextBlocks(message slack.Message) []*slack.RichTextSection {
	richTextSectionBlocks := make([]*slack.RichTextSection, 0)
	for _, block := range message.Blocks.BlockSet {
		switch block.BlockType() {
		case slack.MBTRichText:
			sectionBlock := block.(*slack.RichTextBlock)
			for _, element := range sectionBlock.Elements {
				switch element.RichTextElementType() {
				case slack.RTESection:
					section := element.(*slack.RichTextSection)
					richTextSectionBlocks = append(richTextSectionBlocks, section)

				case slack.RTEList:
					list := element.(*slack.RichTextList)
					for _, item := range list.Elements {
						switch item.RichTextElementType() {
						case slack.RTESection:
							section := item.(*slack.RichTextSection)
							richTextSectionBlocks = append(richTextSectionBlocks, section)
						}
					}
				}
			}
		}
	}

	return richTextSectionBlocks
}

type Config struct {
	SlackToken       string
	SlackCookieValue string
}

func NewSlackClient(config Config) *SlackClient {
	return &SlackClient{
		accessToken: config.SlackToken,
		cookieValue: config.SlackCookieValue,
	}
}

func (c *SlackClient) GetOrderUrls(channelId, timestamp string) ([]LinkWithUser, error) {
	links := make([]LinkWithUser, 0)
	httpClient, err := c.getHttpClient()
	if err != nil {
		log.Printf("Error getting http client: %s\n", err.Error())
		return links, err
	}

	api := slack.New(
		c.accessToken,
		slack.OptionDebug(true),
		slack.OptionHTTPClient(httpClient),
	)

	messages, _, _, err := api.GetConversationReplies(
		&slack.GetConversationRepliesParameters{
			ChannelID:          channelId,
			Timestamp:          timestamp,
			IncludeAllMetadata: true,
		},
	)
	if err != nil {
		log.Printf("Error getting thread messages: %s\n", err.Error())
		return links, err
	}

	userIds := make([]string, 0)

	userLinksMap := make(map[string][]string)

	for _, message := range messages {
		userIds = append(userIds, message.User)

		if _, ok := userLinksMap[message.User]; !ok {
			userLinksMap[message.User] = make([]string, 0)
		}

		richTextBlocks := getRichtextBlocks(message)

		for _, block := range richTextBlocks {
			links := parseLinksFromSection(block)
			userLinksMap[message.User] = append(userLinksMap[message.User], links...)
		}
	}

	users, err := api.GetUsersInfo(userIds...)
	if err != nil {
		return links, err
	}

	if users == nil {
		return links, errors.New("no users found")
	}

	for _, user := range *users {
		name := strings.Join([]string{user.Profile.FirstName, user.Profile.LastName}, " ")
		userLinks := userLinksMap[user.ID]
		for _, link := range userLinks {
			links = append(links, LinkWithUser{
				User: name,
				Link: link,
			})
		}
	}

	return links, nil
}

func parseLinksFromSection(section *slack.RichTextSection) []string {
	links := make([]string, 0)
	for _, sectionElement := range section.Elements {
		switch sectionElement.RichTextSectionElementType() {
		case slack.RTSELink:
			link := sectionElement.(*slack.RichTextSectionLinkElement)
			if !strings.Contains(link.URL, "kofio.cz") {
				continue
			}
			links = append(links, link.URL)
		}
	}

	return links
}
