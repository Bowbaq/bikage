package main

import (
	"flag"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"github.com/PuerkitoBio/goquery"
)

var (
	username = flag.String("u", "", "citibike.com username")
	password = flag.String("p", "", "citibike.com password")
)

func main() {
	flag.Parse()

	var (
		doc *goquery.Document
		err error
	)

	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal(err)
	}

	client := &http.Client{Jar: jar}

	log.Println("Grabbing CSRF token")
	resp, err := client.Get("https://www.citibikenyc.com/login")
	if err != nil {
		log.Fatal(err)
	}

	if doc, err = goquery.NewDocumentFromResponse(resp); err != nil {
		log.Fatal(err)
	}

	csrf, ok := doc.Find("#login-form .hidden input").Attr("value")
	if !ok {
		log.Fatal("couldn't find csrf token")
	}

	log.Println("Logging in")
	resp, err = client.PostForm(
		"https://www.citibikenyc.com/login",
		url.Values{
			"ci_csrf_token":      {csrf},
			"subscriberUsername": {*username},
			"subscriberPassword": {*password},
			"login_submit":       {"Login"},
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	resp.Body.Close()

	log.Println("Navigating to the Trips page")
	resp, err = client.Get("https://www.citibikenyc.com/member/trips")

	if doc, err = goquery.NewDocumentFromResponse(resp); err != nil {
		log.Fatal(err)
	}

	doc.Find("#tripTable .trip").Each(func(i int, tr *goquery.Selection) {
		tr.Children().Each(func(i int, td *goquery.Selection) {
			log.Println(td.Text())
		})
		log.Println("====================")
	})
}
