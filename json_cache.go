package bikage

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"sync"
)

var (
	cache_path = "./distances.json"
)

type JsonCache struct {
	cache map[string]uint64
	sync.RWMutex
}

func NewJsonCache() *JsonCache {
	c := &JsonCache{cache: make(map[string]uint64)}
	c.Lock()
	c.deserialize()
	c.Unlock()

	return c
}

func (c *JsonCache) GetDistance(route Route) (uint64, bool) {
	c.RLock()
	distance, ok := c.cache[make_key(route.From, route.To)]
	c.RUnlock()

	return distance, ok
}

func (c *JsonCache) PutDistance(route Route, distance uint64) {
	c.Lock()

	c.cache[make_key(route.From, route.To)] = distance
	c.serialize()

	c.Unlock()
}

func (c *JsonCache) GetTrip(username, id string) (Trip, bool) {
	return Trip{}, false
}

func (c *JsonCache) PutTrip(trip Trip) {}

func (c *JsonCache) deserialize() {
	data, err := ioutil.ReadFile(cache_path)
	if err != nil {
		log.Println("JsonCache DESERIALIZE error ->", err)
		return
	}

	json.Unmarshal(data, &c.cache)
}

func (c *JsonCache) serialize() {
	data, err := json.MarshalIndent(c.cache, "", "  ")
	if err != nil {
		log.Println(err)
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
