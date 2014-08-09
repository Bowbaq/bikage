package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/Bowbaq/bikage"
)

var (
	username string
	password string

	google_api_key string
	mongo_url      string
)

func init() {
	flag.StringVar(&username, "u", "", "citibike.com username")
	flag.StringVar(&password, "p", "", "citibike.com password")

	flag.StringVar(&google_api_key, "google-api-key", "", "Google API key (directions API)")
	flag.StringVar(&mongo_url, "mongo-url", "", "MongoDB url (persistent distance cache)")
}

func main() {
	flag.Parse()

	bikage, err := bikage.NewBikage(google_api_key, bikage.NewCache(mongo_url))
	if err != nil {
		log.Fatalln(err)
	}

	trips, _ := bikage.GetTrips(username, password)
	stats := bikage.ComputeStats(trips)

	fmt.Println(stats)
}
