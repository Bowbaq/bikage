package bikage

import "log"

type Cache interface {
	GetDistance(route Route) (uint64, bool)
	PutDistance(route Route, distance uint64)

	GetTrip(username, id string) (Trip, bool)
	GetTrips(username string) Trips
	PutTrip(username string, trip Trip)
}

func NewCache(mongo_url string) Cache {
	if mongo_url == "" {
		return NewJsonCache()
	}

	if cache, err := NewMongoCache(mongo_url); err == nil {
		return cache
	} else {
		log.Println("Bikage CACHE error ->", err)
		return NewJsonCache()
	}
}
