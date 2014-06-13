package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"html/template"

	"github.com/gorilla/mux"
)

var (
	username       = flag.String("u", "", "citibike.com username")
	password       = flag.String("p", "", "citibike.com password")

	http_port     = flag.String("http", "", "-http $PORT for HTTP server")

	google_api_key = flag.String("k", "", "Google API key (directions API)")
)

type credentials struct {
	username, password string
}

type distance struct {
	Km_dist, Mi_dist float64
}

func main() {
	flag.Parse()

	distances := NewDistances(*google_api_key)

	stations, err := GetStations()
	if err != nil {
		log.Fatalln(err)
	}

	if *http_port == "" {
		dist, err := compute_distance(credentials{*username, *password}, stations, distances)
		if err != nil {
			log.Fatalln(err)
		}

		fmt.Println("Total distance:", dist.String())
		return
	}

	index := template.Must(template.ParseFiles("./web/home.html"))

	router := mux.NewRouter()

	router.HandleFunc("/", func(res http.ResponseWriter, req *http.Request){
		err := req.ParseForm()
		if err != nil {
			http.Error(res, "malformed request: "+err.Error(), http.StatusBadRequest)
			return
		}

		creds := credentials{
			req.PostFormValue("username"),
			req.PostFormValue("password"),
		}

	  dist, err := compute_distance(creds, stations, distances)
	  index.Execute(res, struct{Distance string}{dist.String()})
	}).Methods("POST")

	router.HandleFunc("/", func(res http.ResponseWriter, req *http.Request){
		index.Execute(res, struct{Distance string}{})
	}).Methods("GET")

	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./web/"))).Methods("GET")

  http.Handle("/", router)
  log.Println("Bikage listening on port", *http_port)
  log.Fatal(http.ListenAndServe(":" + *http_port, nil))
}

func compute_distance(creds credentials, stations map[uint64]Station, distances *Distances) (distance, error) {
	user_trips, err := GetTrips(creds.username, creds.password, stations)
	if err != nil {
		return distance{}, err
	}

	var total uint64
	for _, user_trip := range user_trips {
		dist, err := distances.Get(user_trip.Trip)
		if err != nil {
			log.Println(err)
			continue
		}

		total = total + dist
	}

	km_dist := float64(total) / 1000
	mi_dist := km_dist * 0.621371192

	return distance{km_dist, mi_dist}, nil
}

func (r distance) String() string {
	return fmt.Sprintf("%.1f km (%.1f mi)\n", r.Km_dist, r.Mi_dist)
}
