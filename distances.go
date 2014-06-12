package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

var (
	api_base = "https://maps.googleapis.com/maps/api/directions/json?"

	save_as = "./distances.json"
)

func NewDistances(api_key string) *Distances {
	distances := &Distances{make(map[string]uint64), api_key}
	distances.deserialize()

	return distances
}

func (dist *Distances) Get(trip Trip) (uint64, error) {
	if distance, ok := dist.cache[trip.Key()]; ok {
		return distance, nil
	}

	distance, err := dist.Calculate(trip)
	if err != nil {
		return 0, err
	}

	return distance, nil
}

func (dist *Distances) Calculate(trip Trip) (uint64, error) {
	q := url.Values{}
	q.Add("key", dist.api_key)
	q.Add("origin", trip.From.String())
	q.Add("destination", trip.To.String())
	q.Add("mode", "bicycling")

	resp, err := http.Get(api_base + q.Encode())
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var response DirectionsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return 0, err
	}

	distance, err := response.Distance()
	if err != nil {
		return 0, err
	}

	dist.cache[trip.Key()] = distance
	dist.serialize()

	return distance, nil
}

func (dist *Distances) deserialize() {
	data, err := ioutil.ReadFile(save_as)
	if err != nil {
		log.Println(err)
		return
	}

	json.Unmarshal(data, &dist.cache)
}

func (dist *Distances) serialize() {
	data, err := json.MarshalIndent(dist.cache, "", "  ")
	if err != nil {
		log.Println(err)
		return
	}

	err = ioutil.WriteFile(save_as, data, 0755)
	if err != nil {
		log.Println(err)
	}
}

func (coord Coord) String() string {
	return fmt.Sprintf("%.8f,%.8f", coord.Lat, coord.Lng)
}

func (trip Trip) Key() string {
	return fmt.Sprintf("%d,%d", trip.From.Id, trip.To.Id)
}

func (response DirectionsResponse) Distance() (uint64, error) {
	if len(response.Routes) > 0 {
		route := response.Routes[0]
		if len(route.Legs) > 0 {
			return route.Legs[0].Distance.Value, nil
		} else {
			return 0, errors.New("no legs in route")
		}
	} else {
		return 0, errors.New("no route")
	}
}
