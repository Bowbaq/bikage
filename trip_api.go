package bikage

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	login_page = "https://www.citibikenyc.com/login"
	trips_page = "https://www.citibikenyc.com/member/trips"
)

type TripAPI interface {
	WithCache(cache TripCache) TripAPI

	GetTrips(username, password string) (Trips, error)
	GetCachedTrips(username string) Trips
}

type trip_api struct {
	cache    TripCache
	stations Stations
}

func NewTripAPI(stations Stations) TripAPI {
	return &trip_api{
		cache:    new(NoopCache),
		stations: stations,
	}
}

func (ta *trip_api) WithCache(cache TripCache) TripAPI {
	ta.cache = cache
	return ta
}

func (tc *trip_api) GetTrips(username, password string) (Trips, error) {
	citibike, err := new_citibike(username, password)
	if err != nil {
		return nil, err
	}

	return citibike.get_all_trips(username, tc.cache, tc.stations)
}

func (tc *trip_api) GetCachedTrips(username string) Trips {
	return tc.cache.GetTrips(username)
}

type citibike struct {
	http *http.Client
	csrf string
}

func new_citibike(username, password string) (*citibike, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	cb := citibike{
		http: &http.Client{Jar: jar},
	}

	err = cb.get_csrf()
	if err != nil {
		return nil, err
	}

	err = cb.login(username, password)
	if err != nil {
		return nil, err
	}

	return &cb, nil
}

func (cb *citibike) get_csrf() error {
	resp, err := cb.http.Get(login_page)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return err
	}

	csrf, ok := doc.Find("#login-form .hidden input").Attr("value")
	if !ok {
		return errors.New("couldn't find csrf token")
	}

	cb.csrf = csrf

	return nil
}

func (cb *citibike) login(username, password string) error {
	resp, err := cb.http.PostForm(login_page, url.Values{
		"ci_csrf_token":      {cb.csrf},
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

func (cb *citibike) get_trips(url string, stations Stations) (Trips, string, error) {
	resp, err := cb.http.Get(url)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return nil, "", err
	}

	var trips Trips
	doc.Find("#tripTable .trip").Each(func(i int, tr *goquery.Selection) {
		trip_id, ok := tr.Attr("id")
		if !ok {
			return
		}

		start_station, err := parse_station(tr, stations, "data-start-station-id")
		if err != nil {
			log.Println("COULDN'T PARSE STATION", err)
			return
		}

		end_station, err := parse_station(tr, stations, "data-end-station-id")
		if err != nil {
			log.Println("COULDN'T PARSE STATION", err)
			return
		}

		start_time, err := parse_time(tr, "data-start-timestamp")
		if err != nil {
			return
		}

		end_time, err := parse_time(tr, "data-end-timestamp")
		if err != nil {
			return
		}

		trip := Trip{
			Id: trip_id,
			Route: Route{
				From: start_station,
				To:   end_station,
			},
			StartedAt: *start_time,
			EndedAt:   *end_time,
		}

		trips = append(trips, trip)
	})

	nav := doc.Find("nav.pagination a").FilterFunction(func(i int, link *goquery.Selection) bool {
		return link.Text() == ">"
	})

	if nav.Size() == 0 {
		return trips, "", nil
	}

	next_page, ok := nav.Last().Attr("href")
	if !ok {
		return trips, "", nil
	}

	return trips, next_page, nil
}

func (cb *citibike) get_all_trips(username string, cache TripCache, stations Stations) (Trips, error) {
	var fetch func(string) (Trips, error)
	fetch = func(page string) (Trips, error) {
		trips, next_page, err := cb.get_trips(page, stations)
		if err != nil {
			return nil, err
		}

		for _, trip := range trips {
			cache.PutTrip(username, trip)
		}

		if next_page != "" {
			next_trips, err := fetch(next_page)
			return append(trips, next_trips...), err
		}

		return trips, nil
	}

	trips, err := fetch(trips_page)
	sort.Sort(trips)

	return trips, err
}

func parse_station(node *goquery.Selection, stations Stations, id_attr string) (Station, error) {
	id, err := parse_uint64(node, id_attr)
	if err != nil {
		return Station{}, err
	}

	station, found := stations[id]
	if !found {
		index := 1
		if strings.Contains(id_attr, "end") {
			index += 2
		}
		missing := node.Children().Eq(index).Text()

		return Station{}, fmt.Errorf("Unknown station id: %d, %s", id, missing)
	}

	return station, nil
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
