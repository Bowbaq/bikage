package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Bowbaq/bikage"
)

var (
	username string
	password string

	google_api_key string
	mongo_url      string
)

func init() {
	flag.StringVar(&username, "u", "", "citibike.com username (required)")
	flag.StringVar(&password, "p", "", "citibike.com password (required)")

	flag.StringVar(&google_api_key, "google-api-key", "", "Google API key, directions API must be enabled (required)")
	flag.StringVar(&mongo_url, "mongo-url", "", "MongoDB url (persistent distance cache) (optional, defaults to local JSON cache)")
}

func main() {
	flag.Parse()

	if username == "" || password == "" || google_api_key == "" {
		flag.Usage()
		os.Exit(1)
	}

	bikage, err := bikage.NewBikage(google_api_key, bikage.NewCache(mongo_url))
	if err != nil {
		log.Fatalln(err)
	}

	trips, err := bikage.GetTrips(username, password)
	if err != nil {
		log.Fatalln(err)
	}

	stats := bikage.ComputeStats(trips)

	fmt.Println(stats)
}
