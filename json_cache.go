package bikage

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"sync"
)

var (
	distance_path = "./distances.json"
	trip_path     = "./trips.json"
)

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
	distance, ok := c.distances[make_key(route.From, route.To)]
	c.RUnlock()

	return distance, ok
}

func (c *JsonCache) PutDistance(route Route, distance uint64) {
	c.Lock()

	c.distances[make_key(route.From, route.To)] = distance
	c.serialize()

	c.Unlock()
}

func (c *JsonCache) GetTrip(username, id string) (Trip, bool) {
	return Trip{}, false

	user_trips, hit := c.trips[username]
	if !hit {
		return Trip{}, false
	}

	trip, hit := user_trips[id]

	return trip, hit
}

func (c *JsonCache) GetTrips(username string) Trips {
	return Trips{}

	trips := make(Trips, 0)

	user_trips, hit := c.trips[username]
	if !hit {
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

	_, hit := c.trips[username]
	if !hit {
		c.trips[username] = make(map[string]Trip)
	}

	c.trips[username][trip.Id] = trip
	c.serialize()

	c.Unlock()
}

func (c *JsonCache) deserialize() {
	data, err := ioutil.ReadFile(distance_path)
	if err != nil {
		log.Println("JsonCache DESERIALIZE error ->", err)
		return
	}

	json.Unmarshal(data, &c.distances)

	data, err = ioutil.ReadFile(trip_path)
	if err != nil {
		log.Println("JsonCache DESERIALIZE error ->", err)
		return
	}

	json.Unmarshal(data, &c.trips)
}

func (c *JsonCache) serialize() {
	data, err := json.MarshalIndent(c.distances, "", "  ")
	if err != nil {
		log.Println(err)
		return
	}

	err = ioutil.WriteFile(distance_path, data, 0755)
	if err != nil {
		log.Println("JsonCache SERIALIZE error ->", err)
	}

	data, err = json.MarshalIndent(c.trips, "", "  ")
	if err != nil {
		log.Println(err)
		return
	}

	err = ioutil.WriteFile(trip_path, data, 0755)
	if err != nil {
		log.Println("JsonCache SERIALIZE error ->", err)
	}
}

func make_key(from, to Station) string {
	return fmt.Sprintf("%d,%d", from.Id, to.Id)
}
