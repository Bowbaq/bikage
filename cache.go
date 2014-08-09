package bikage

import "log"

type Cache interface {
	Get(trip Trip) (uint64, bool)
	Put(trip Trip, distance uint64)
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
