package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

var (
	api_base = "https://maps.googleapis.com/maps/api/directions/json?"
)

type directionsResponse struct {
	Routes []struct {
		Legs []struct {
			Distance struct {
				Text  string
				Value uint64
			}
		}
	}
}

func NewDistanceCache(api_key string, cache Cache) *DistanceCache {
	return &DistanceCache{
		cache:   cache,
		api_key: api_key,
	}
}

func (dist *DistanceCache) Get(trip Trip) (uint64, error) {
	distance, ok := dist.cache.Get(trip)
	if ok {
		return distance, nil
	}

	distance, err := dist.Calculate(trip)
	if err != nil {
		return 0, err
	}

	return distance, nil
}

func (dist *DistanceCache) Calculate(trip Trip) (uint64, error) {
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

	var response directionsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return 0, err
	}

	distance, err := response.distance()
	if err != nil {
		return 0, err
	}

	dist.cache.Put(trip, distance)

	return distance, nil
}

func (coord Coord) String() string {
	return fmt.Sprintf("%.8f,%.8f", coord.Lat, coord.Lng)
}

func (response directionsResponse) distance() (uint64, error) {
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
