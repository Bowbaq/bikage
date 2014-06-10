package main

import (
	"errors"
	"flag"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var (
	username = flag.String("u", "", "citibike.com username")
	password = flag.String("p", "", "citibike.com password")
)

type Station struct {
	Id       string
	Label    string
	Lat, Lng float32
}

type Trip struct {
	Id        string
	StartId   string
	StartTime time.Time
	EndId     string
	EndTime   time.Time
}

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
		id, _ := tr.Attr("id")
		start_id, _ := tr.Attr("data-start-station-id")
		start_time, _ := parse_time(tr, "data-start-timestamp")
		end_id, _ := tr.Attr("data-end-station-id")
		end_time, _ := parse_time(tr, "data-end-timestamp")

		trip := Trip{id, start_id, *start_time, end_id, *end_time}
		log.Println(trip)
	})
}

func parse_time(node *goquery.Selection, attr string) (*time.Time, error) {
	time_str, ok := node.Attr(attr)
	if !ok {
		return nil, errors.New("attribute " + attr + " does not exist")
	}

	time_sec, err := strconv.ParseInt(time_str, 10, 64)
	if err != nil {
		return nil, err
	}

	time := time.Unix(time_sec, 0)
	return &time, nil
}
