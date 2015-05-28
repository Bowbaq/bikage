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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Bowbaq/pool"
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

func (ta *trip_api) GetTrips(username, password string) (Trips, error) {
	citibike, err := new_citibike(username, password, &ta.stations)
	if err != nil {
		return nil, err
	}

	return citibike.get_all_trips(username, ta.cache)
}

func (ta *trip_api) GetCachedTrips(username string) Trips {
	return ta.cache.GetTrips(username)
}

type citibike struct {
	http       *http.Client
	stations   *Stations
	trips_path string
}

func new_citibike(username, password string, stations *Stations) (*citibike, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	cb := citibike{
		http:     &http.Client{Jar: jar},
		stations: stations,
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

type fetchTrips struct {
	wg   *sync.WaitGroup
	page int
	cb   *citibike
}

var tripPool = pool.NewRateLimitedPool(10, 20, 10, func(id uint, payload interface{}) interface{} {
	job := payload.(fetchTrips)
	defer job.wg.Done()

	doc, err := job.cb.get_trips_document(job.cb.trips_path + "?pageNumber=" + strconv.Itoa(job.page))
	if err != nil {
		return err
	}

	return job.cb.parse_trips(doc)
})

func (cb *citibike) get_all_trips(username string, cache TripCache) (Trips, error) {
	doc, err := cb.get_trips_document(cb.trips_path)
	if err != nil {
		return nil, err
	}

	next_page, last_page := parse_navigation(doc, cb.trips_path)

	var wg sync.WaitGroup
	wg.Add(last_page - next_page + 1)

	var jobs []pool.Job
	for p := next_page; p <= last_page; p++ {
		job := pool.NewJob(fetchTrips{&wg, p, cb})
		tripPool.Submit(job)
		jobs = append(jobs, job)
	}
	wg.Wait()

	trips := cb.parse_trips(doc)
	for _, job := range jobs {
		result := job.Result()
		switch result.(type) {
		case Trips:
			trips = append(trips, result.(Trips)...)
		case error:
			return trips, result.(error)
		default:
			return trips, errors.New("Unexpected return type from the pool")
		}
	}

	sort.Sort(trips)
	for _, trip := range trips {
		cache.PutTrip(username, trip)
	}

	return trips, nil
}

func (cb *citibike) get_trips_document(path string) (*goquery.Document, error) {
	resp, err := cb.http.Get("https://member.citibikenyc.com" + path)
	if err != nil {
		return nil, err
	}

	return goquery.NewDocumentFromResponse(resp)
}

func parse_navigation(doc *goquery.Document, trips_path string) (int, int) {
	next, nok := doc.Find(".ed-paginated-navigation__pages-group__link_next").Attr("href")
	last, lok := doc.Find(".ed-paginated-navigation__pages-group__link_last").Attr("href")
	if !nok || !lok {
		return 1, 1
	}

	next_str := strings.TrimPrefix(next, trips_path+"?pageNumber=")
	last_str := strings.TrimPrefix(last, trips_path+"?pageNumber=")

	next_page, nerr := strconv.Atoi(next_str)
	last_page, lerr := strconv.Atoi(last_str)
	if nerr != nil || lerr != nil {
		return 1, 1
	}

	return next_page, last_page
}

func (cb *citibike) parse_trips(doc *goquery.Document) Trips {
	var trips Trips

	doc.Find(".ed-table__items .ed-table__item_trip").Each(func(i int, tr *goquery.Selection) {
		start_station, err := cb.parse_station(tr, ".ed-table__item__info__sub-info_trip-start-station")
		if err != nil {
			log.Println("COULDN'T PARSE START STATION", err)
			return
		}

		end_station, err := cb.parse_station(tr, ".ed-table__item__info__sub-info_trip-end-station")
		if err != nil {
			log.Println("COULDN'T PARSE END STATION", err)
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

	return trips
}

func (cb *citibike) parse_station(node *goquery.Selection, name_div string) (Station, error) {
	station_label := node.Find(name_div).Text()
	if station, ok := (*cb.stations)[station_label]; ok {
		return station, nil
	}

	return Station{}, fmt.Errorf("Unknown station: %s", station_label)
}

func parse_time(node *goquery.Selection, time_div string) (time.Time, error) {
	return time.Parse("01/02/2006 3:04:05 PM", node.Find(time_div).Text())
}
