package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

var (
	bikage Bikage
)

type Bikage struct {
	// CLI params
	creds          credentials
	http_port      string
	google_api_key string
	mongo_url      string

	distance_cache *DistanceCache
	stations       Stations
}

type credentials struct {
	username, password string
}

type distance struct {
	Km_dist, Mi_dist float64
}

func init() {
	flag.StringVar(&bikage.creds.username, "u", "", "citibike.com username")
	flag.StringVar(&bikage.creds.password, "p", "", "citibike.com password")

	flag.StringVar(&bikage.http_port, "http", "", "-http $PORT for HTTP server")

	flag.StringVar(&bikage.google_api_key, "k", "", "Google API key (directions API)")
	flag.StringVar(&bikage.mongo_url, "mgo", "", "MongoDB url (persistent distance cache)")
}

func main() {
	flag.Parse()

	bikage.Init()
	bikage.Run()
}

func (bk *Bikage) Init() {
	if bk.mongo_url == "" {
		bk.distance_cache = NewDistanceCache(bk.google_api_key, NewJsonCache())
	} else {
		var cache Cache

		cache, err := NewMongoCache(bk.mongo_url)
		if err != nil {
			log.Println("Bikage CACHE error ->", err)
			cache = NewJsonCache()
		}
		bk.distance_cache = NewDistanceCache(bk.google_api_key, cache)
	}

	stations, err := GetStations()
	if err != nil {
		log.Fatalln("Bikage STATIONS GET error ->", err)
	}
	bk.stations = stations
}

func (bk *Bikage) Run() {
	if bk.http_port == "" {
		bk.CliRun()
	} else {
		bk.ServeHTTP()
	}
}

func (bk *Bikage) CliRun() {
	dist, err := bk.compute_distance(bk.creds)
	if err != nil {
		log.Fatalln("Bikage CALC DISTANCE error ->", err)
	}

	fmt.Println("Total distance:", dist.String())
	return
}

func (bk *Bikage) ServeHTTP() {
	index := template.Must(template.ParseFiles("./web/home.html"))

	router := mux.NewRouter()
	router.HandleFunc("/", bk.IndexHandler(index)).Methods("GET")
	router.HandleFunc("/", bk.DistanceHandler(index)).Methods("POST")
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./web/"))).Methods("GET")

	http.Handle("/", router)
	log.Println("Bikage listening on port", bk.http_port)
	log.Fatal(http.ListenAndServe(":"+bk.http_port, nil))
}

func (bk *Bikage) IndexHandler(index *template.Template) func(http.ResponseWriter, *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		index.Execute(res, struct{ Distance string }{})
	}
}

func (bk *Bikage) DistanceHandler(index *template.Template) func(http.ResponseWriter, *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		err := req.ParseForm()
		if err != nil {
			http.Error(res, "Bikage MALFORMED REQUEST error -> "+err.Error(), http.StatusBadRequest)
			return
		}

		creds := credentials{
			req.PostFormValue("username"),
			req.PostFormValue("password"),
		}

		dist, err := bk.compute_distance(creds)
		if err != nil {
			http.Error(res, "Bikage CALC DISTANCE error -> "+err.Error(), http.StatusInternalServerError)
			return
		}

		index.Execute(res, struct{ Distance string }{dist.String()})
	}

}

func (bk *Bikage) compute_distance(creds credentials) (distance, error) {
	user_trips, err := GetTrips(creds.username, creds.password, bk.stations)
	if err != nil {
		return distance{}, err
	}

	var total uint64
	for _, user_trip := range user_trips {
		dist, err := bk.distance_cache.Get(user_trip.Trip)
		if err != nil {
			log.Println("Bikage GET TRIP DISTANCE error -> ", err)
			continue
		}

		total = total + dist
	}

	km_dist := float64(total) / 1000
	mi_dist := km_dist * 0.621371192

	return distance{km_dist, mi_dist}, nil
}

func (r distance) String() string {
	return fmt.Sprintf("%.1f km (%.1f mi)", r.Km_dist, r.Mi_dist)
}
