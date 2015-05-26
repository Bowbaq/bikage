package bikage

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sort"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	login_form     = "https://member.citibikenyc.com/profile/login"
	login_endpoint = "https://member.citibikenyc.com/profile/login_check"
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
	http       *http.Client
	trips_path string
}

func new_citibike(username, password string) (*citibike, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	cb := citibike{
		http: &http.Client{Jar: jar},
	}

	csrf, err := cb.get_csrf()

	err = cb.login(username, password, csrf)
	if err != nil {
		return nil, err
	}

	return &cb, nil
}

func (cb *citibike) get_csrf() (string, error) {
	resp, err := cb.http.Get(login_form)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return "", err
	}

	csrf, ok := doc.Find(`#loginPopupId input[name="_login_csrf_security_token"]`).Attr("value")
	if !ok {
		return "", errors.New("couldn't find csrf token")
	}

	return csrf, nil
}

func (cb *citibike) login(username, password, csrf string) error {
	resp, err := cb.http.PostForm(login_endpoint, url.Values{
		"_username":                  {username},
		"_password":                  {password},
		"_login_csrf_security_token": {csrf},
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return err
	}

	trips_path, ok := doc.Find(".ed-profile-menu__link_trips a").Attr("href")
	if !ok {
		return errors.New("couldn't find trips page link")
	}
	cb.trips_path = trips_path

	return nil
}

func (cb *citibike) get_trips(path string, stations Stations) (Trips, string, error) {
	resp, err := cb.http.Get("https://member.citibikenyc.com" + path)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return nil, "", err
	}

	var trips Trips
	doc.Find(".ed-table__items .ed-table__item_trip").Each(func(i int, tr *goquery.Selection) {
		start_station, err := parse_station(tr, stations, ".ed-table__item__info__sub-info_trip-start-station")
		if err != nil {
			log.Println("COULDN'T PARSE START STATION", err, path)
			return
		}

		end_station, err := parse_station(tr, stations, ".ed-table__item__info__sub-info_trip-end-station")
		if err != nil {
			log.Println("COULDN'T PARSE END STATION", err, path)
			return
		}

		start_time, err := parse_time(tr, ".ed-table__item__info__sub-info_trip-start-date")
		if err != nil {
			return
		}

		end_time, err := parse_time(tr, ".ed-table__item__info__sub-info_trip-end-date")
		if err != nil {
			return
		}

		trip := Trip{
			Id: fmt.Sprintf("%x", sha1.Sum([]byte(fmt.Sprintf("%s-%s-%s-%s", start_station.Label, end_station.Label, start_time, end_time)))),
			Route: Route{
				From: start_station,
				To:   end_station,
			},
			StartedAt: start_time,
			EndedAt:   end_time,
		}

		trips = append(trips, trip)
	})

	nav := doc.Find(".ed-paginated-navigation__container a").FilterFunction(func(i int, link *goquery.Selection) bool {
		return link.Text() == "Older"
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
			log.Println("Caching", trip)
			cache.PutTrip(username, trip)
		}

		if next_page != "" {
			next_trips, err := fetch(next_page)
			return append(trips, next_trips...), err
		}

		return trips, nil
	}

	trips, err := fetch(cb.trips_path)
	sort.Sort(trips)

	return trips, err
}

func parse_station(node *goquery.Selection, stations Stations, name_div string) (Station, error) {
	station_label := node.Find(name_div).Text()
	if station, ok := stations[station_label]; ok {
		return station, nil
	}

	return Station{}, fmt.Errorf("Unknown station: %s", station_label)
}

func parse_time(node *goquery.Selection, time_div string) (time.Time, error) {
	return time.Parse("01/02/2006 3:04:05 PM", node.Find(time_div).Text())
}
