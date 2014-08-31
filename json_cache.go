package bikage

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"sync"
)

const cache_path = "./bikage_cache.json"

type JsonCache struct {
	distances map[string]uint64
	trips     map[string]map[string]Trip
	sync.RWMutex
}

func NewJsonCache() *JsonCache {
	c := &JsonCache{
		distances: make(map[string]uint64),
		trips:     make(map[string]map[string]Trip),
	}

	c.Lock()
	c.deserialize()
	c.Unlock()

	return c
}

func (c *JsonCache) GetDistance(route Route) (uint64, bool) {
	c.RLock()
	distance, found := c.distances[make_key(route.From, route.To)]
	c.RUnlock()

	return distance, found
}

func (c *JsonCache) PutDistance(route Route, distance uint64) {
	c.Lock()

	c.distances[make_key(route.From, route.To)] = distance
	c.serialize()

	c.Unlock()
}

func (c *JsonCache) GetTrip(username, id string) (Trip, bool) {
	c.RLock()
	user_trips, found := c.trips[username]
	c.RUnlock()

	if !found {
		return Trip{}, false
	}

	trip, found := user_trips[id]

	return trip, found
}

func (c *JsonCache) GetTrips(username string) Trips {
	trips := make(Trips, 0)

	c.RLock()
	user_trips, found := c.trips[username]
	c.RUnlock()

	if !found {
		return trips
	}

	for _, trip := range user_trips {
		trips = append(trips, trip)
	}

	sort.Sort(trips)

	return trips
}

func (c *JsonCache) PutTrip(username string, trip Trip) {
	c.Lock()

	_, found := c.trips[username]
	if !found {
		c.trips[username] = make(map[string]Trip)
	}

	c.trips[username][trip.Id] = trip
	c.serialize()

	c.Unlock()
}

type serialized struct {
	Distances map[string]uint64
	Trips     map[string]map[string]Trip
}

func (c *JsonCache) deserialize() {
	data, err := ioutil.ReadFile(cache_path)
	if err != nil {
		log.Println("JsonCache READ error ->", err)
		return
	}

	var cache serialized
	json.Unmarshal(data, &cache)

	c.distances = cache.Distances
	c.trips = cache.Trips
}

func (c *JsonCache) serialize() {
	cache := serialized{c.distances, c.trips}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		log.Println("JsonCache MARSHALL error ->", err)
		return
	}

	err = ioutil.WriteFile(cache_path, data, 0755)
	if err != nil {
		log.Println("JsonCache SERIALIZE error ->", err)
	}
}

func make_key(from, to Station) string {
	return fmt.Sprintf("%d,%d", from.Id, to.Id)
}
