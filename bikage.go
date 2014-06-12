package main

import (
	"flag"
	"log"
	"fmt"
)

var (
	username       = flag.String("u", "", "citibike.com username")
	password       = flag.String("p", "", "citibike.com password")
	google_api_key = flag.String("k", "", "Google API key (directions API)")
)

func main() {
	flag.Parse()

	distances := NewDistances(*google_api_key)

	stations, err := GetStations()
	if err != nil {
		log.Fatalln(err)
	}

	user_trips, err := GetTrips(*username, *password, stations)
	if err != nil {
		log.Fatalln(err)
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

	km_dist := float64(total)/1000
	mi_dist := km_dist * 0.621371192

	fmt.Printf("Total distance: %.1f km (%.1f mi)\n", km_dist, mi_dist)
}
