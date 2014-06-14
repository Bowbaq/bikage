package main

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
	cache := &JsonCache{cache: make(map[string]uint64)}
	cache.deserialize()

	return cache
}

func (c *JsonCache) Get(trip Trip) (uint64, bool) {
	c.RLock()
	dist, ok := c.cache[make_key(trip)]
	c.RUnlock()

	return dist, ok
}

func (c *JsonCache) Put(trip Trip, distance uint64) {
	c.Lock()
	c.cache[make_key(trip)] = distance
	c.Unlock()

	c.serialize()
}

func (c *JsonCache) deserialize() {
	c.Lock()
	defer c.Unlock()

	data, err := ioutil.ReadFile(cache_path)
	if err != nil {
		log.Println("JsonCache DESERIALIZE error ->", err)
		return
	}

	json.Unmarshal(data, &c.cache)
}

func (c *JsonCache) serialize() {
	c.Lock()
	defer c.Unlock()

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

func make_key(trip Trip) string {
	return fmt.Sprintf("%d,%d", trip.From.Id, trip.To.Id)
}
