package parser

import (
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/gocolly/colly/v2"
)

const (
	imageSelector = ".product_profile_picture:not(.a-center)"
	priceSelector = ".price_selector"
	kofioURL      = "https://www.kofio.cz/"
	titleSelector = ".product_page_heading_h1"
)

type Parser struct {
	cookieJar  *cookiejar.Jar
	httpClient *http.Client
	Credentials
	cookies []*http.Cookie
}

type Credentials struct {
	Username string
	Password string
}

func (p *Parser) getCollector() *colly.Collector {
	collector := colly.NewCollector(colly.Async(true))
	collector.Init()
	collector.SetClient(p.httpClient)
	return collector
}

func NewParser(credentials Credentials) *Parser {
	return &Parser{cookieJar: nil, Credentials: credentials}
}

func (p *Parser) Init() error {
	kofioUrl, err := url.Parse(kofioURL)
	if err != nil {
		return err
	}
	cookieJar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}

	httpClient := &http.Client{
		Jar: cookieJar,
	}

	csrfRes, err := httpClient.Get("https://www.kofio.cz/auth/login")
	if err != nil {
		panic(err)
	}

	cookies := csrfRes.Cookies()

	csrfVal := ""
	for _, cook := range cookies {
		if cook.Name == "csrf_cookie_name" {
			csrfVal = cook.Value
		}
	}

	loginForm := url.Values{}
	loginForm.Add("login", p.Username)
	loginForm.Add("password", p.Password)
	loginForm.Add("csrf_test_name", csrfVal)

	loginUrl, err := url.Parse("https://www.kofio.cz/auth/login")
	if err != nil {
		panic(err)
	}
	csrfHeader := http.Header{}
	csrfHeader.Add("Cookie", "csrf_cookie_name="+csrfVal)
	csrfHeader.Add("Content-Type", "application/x-www-form-urlencoded")
	payload := loginForm.Encode()
	_, err = httpClient.Do(&http.Request{
		Method:           "POST",
		URL:              loginUrl,
		Header:           csrfHeader,
		ContentLength:    0,
		TransferEncoding: []string{},
		Body:             io.NopCloser(strings.NewReader(payload)),
	})
	if err != nil {
		log.Println("ERROR in client do", err)
		panic(err)
	}
	res, err := httpClient.Post(
		"https://www.kofio.cz/auth/login?back=",
		"application/x-www-form-urlencoded",
		strings.NewReader(loginForm.Encode()),
	)
	if err != nil {
		panic(err)
	}

	_, err = io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	res.Body.Close()

	p.cookieJar = cookieJar

	p.cookies = cookieJar.Cookies(kofioUrl)

	p.httpClient = httpClient

	return nil
}

type ImagePrice struct {
	Image string
	Price string
	Title string
}

func (p *Parser) GetImage(productUrl string) (*ImagePrice, error) {
	collector := p.getCollector()

	var imgPrice ImagePrice
	collector.OnHTML(imageSelector, func(h *colly.HTMLElement) {
		imgPrice.Image = getImageSrc(h)
	})

	collector.OnHTML(priceSelector, func(h *colly.HTMLElement) {
		imgPrice.Price = getFirstElementText(h)
	})

	collector.OnHTML(titleSelector, func(h *colly.HTMLElement) {
		imgPrice.Title = getFirstElementText(h)
	})

	err := collector.Visit(productUrl)
	if err != nil {
		return nil, err
	}

	collector.Wait()

	return &imgPrice, nil
}

// price_selector
func getImageSrc(h *colly.HTMLElement) string {
	attrs := h.DOM.First().Children().First().Children().Nodes[0].Attr

	for _, attr := range attrs {
		if attr.Key == "src" {
			return attr.Val
		}
	}

	return ""
}

func getFirstElementText(h *colly.HTMLElement) string {
	price := h.DOM.First().Text()
	return strings.TrimSpace(price)
}
