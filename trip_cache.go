package bikage

import (
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	login_page = "https://www.citibikenyc.com/login"
	trips_page = "https://www.citibikenyc.com/member/trips"
)

type TripCache struct {
	cache    Cache
	stations Stations
}

func NewTripCache(cache Cache, stations Stations) *TripCache {
	return &TripCache{
		cache:    cache,
		stations: stations,
	}
}

func (tc *TripCache) GetTrips(username, password string) (Trips, error) {
	citibike, err := new_citibike(username, password)
	if err != nil {
		return nil, err
	}

	cached_trips := tc.cache.GetTrips(username)

	if len(cached_trips) == 0 {
		return citibike.get_all_trips(username, tc.cache, tc.stations)
	}

	last_cached_trip := cached_trips[len(cached_trips)-1]

	var fetch func(string) (Trips, error)
	fetch = func(page string) (Trips, error) {
		fetched_trips := make(Trips, 0)

		trips, next_page, err := citibike.get_trips(page, tc.stations)
		if err != nil {
			return nil, err
		}

		for _, trip := range trips {
			if trip.Id == last_cached_trip.Id {
				return fetched_trips, nil
			}

			tc.cache.PutTrip(username, trip)
			fetched_trips = append(fetched_trips, trip)
		}

		if next_page != "" {
			next_trips, err := fetch(next_page)
			return append(fetched_trips, next_trips...), err
		}

		return fetched_trips, nil
	}

	new_trips, err := fetch(trips_page)
	if err != nil {
		return nil, err
	}

	cached_trips = append(cached_trips, new_trips...)
	sort.Sort(cached_trips)

	return cached_trips, nil
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

func (cb *citibike) get_all_trips(username string, cache Cache, stations Stations) (Trips, error) {
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