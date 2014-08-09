package bikage

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	login_page = "https://www.citibikenyc.com/login"
	trips_page = "https://www.citibikenyc.com/member/trips"
)

type Trip struct {
	From Station
	To   Station
}

func (t Trip) String() string {
	return fmt.Sprintf("%s -> %s", t.From, t.To)
}

type UserTrip struct {
	Trip
	StartedAt time.Time
	EndedAt   time.Time
}

func (ut UserTrip) String() string {
	return fmt.Sprintf("(%s, start: %s, end: %s)", ut.Trip, ut.StartedAt.Format("Jan 02, 15:04"), ut.EndedAt.Format("Jan 02, 15:04"))
}

func get_trips(username, password string, stations Stations) ([]UserTrip, error) {
	client, err := http_client()
	if err != nil {
		return nil, err
	}

	csrf, err := get_csrf_token(client)
	if err != nil {
		return nil, err
	}

	err = login(client, username, password, csrf)
	if err != nil {
		return nil, err
	}

	return extract_trips(trips_page, client, stations)
}

func http_client() (*http.Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	return &http.Client{Jar: jar}, nil
}

func get_csrf_token(client *http.Client) (string, error) {
	resp, err := client.Get(login_page)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return "", err
	}

	csrf, ok := doc.Find("#login-form .hidden input").Attr("value")
	if !ok {
		return "", errors.New("couldn't find csrf token")
	}

	return csrf, nil
}

func login(client *http.Client, username, password, csrf string) error {
	resp, err := client.PostForm(login_page, url.Values{
		"ci_csrf_token":      {csrf},
		"subscriberUsername": {username},
		"subscriberPassword": {password},
		"login_submit":       {"Login"},
	})
	if err != nil {
		return err
	}

	resp.Body.Close()
	return nil
}

func extract_trips(url string, client *http.Client, stations Stations) ([]UserTrip, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return nil, err
	}

	var trips []UserTrip
	doc.Find("#tripTable .trip").Each(func(i int, tr *goquery.Selection) {
		start_id, err := parse_uint64(tr, "data-start-station-id")
		if err != nil {
			return
		}
		start_station := stations[start_id]

		end_id, err := parse_uint64(tr, "data-end-station-id")
		if err != nil {
			return
		}
		end_station := stations[end_id]

		trip := Trip{start_station, end_station}

		start_time, err := parse_time(tr, "data-start-timestamp")
		if err != nil {
			return
		}
		end_time, err := parse_time(tr, "data-end-timestamp")
		if err != nil {
			return
		}

		user_trip := UserTrip{trip, *start_time, *end_time}

		trips = append(trips, user_trip)
	})

	nav := doc.Find("nav.pagination a").FilterFunction(func(i int, link *goquery.Selection) bool {
		return link.Text() == ">"
	})

	if nav.Size() == 0 {
		return trips, nil
	}

	next_page, ok := nav.Last().Attr("href")
	if !ok {
		return trips, nil
	}

	older_trips, err := extract_trips(next_page, client, stations)
	return append(older_trips, trips...), err
}

func parse_uint64(node *goquery.Selection, attr string) (uint64, error) {
	uint64_str, ok := node.Attr(attr)
	if !ok {
		return 0, errors.New("attribute " + attr + " does not exist")
	}

	return strconv.ParseUint(uint64_str, 10, 64)
}

func parse_time(node *goquery.Selection, attr string) (*time.Time, error) {
	time_sec, err := parse_uint64(node, attr)
	if err != nil {
		return nil, err
	}

	time := time.Unix(int64(time_sec), 0)
	return &time, nil
}
