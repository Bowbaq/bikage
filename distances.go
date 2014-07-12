package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"
)

const (
	api_base            = "https://maps.googleapis.com/maps/api/directions/json?"
	requests_per_second = 8
	api_workers         = 10
)

type DistanceCache struct {
	cache Cache
	api   DistanceAPI
}

func NewDistanceCache(api_key string, cache Cache) *DistanceCache {
	return &DistanceCache{
		cache: cache,
		api:   NewDistanceAPI(api_key),
	}
}

func (dist *DistanceCache) Get(trip UserTrip) (uint64, error) {
	distance, ok := dist.cache.Get(trip.Trip)
	if ok {
		return distance, nil
	}

	distance, err := dist.Calculate(trip)
	if err != nil {
		return 0, err
	}

	return distance, nil
}

func (dist *DistanceCache) GetAll(trips []UserTrip) map[UserTrip]uint64 {
	result := make(map[UserTrip]uint64)

	var misses []UserTrip
	for _, trip := range trips {
		if distance, ok := dist.cache.Get(trip.Trip); ok {
			result[trip] = distance
		} else {
			misses = append(misses, trip)
		}
	}

	for k, v := range dist.CalculateAll(misses) {
		result[k] = v
	}

	return result
}

func (dist *DistanceCache) Calculate(trip UserTrip) (uint64, error) {
	req := distance_request{trip, make(chan distance_response, 1)}
	dist.api.requests <- req
	resp := <-req.result

	if resp.error == nil {
		dist.cache.Put(trip.Trip, resp.distance)
	}

	return resp.distance, resp.error
}

func (dist *DistanceCache) CalculateAll(trips []UserTrip) map[UserTrip]uint64 {
	result := make(map[UserTrip]uint64)

	var requests []distance_request
	for _, trip := range trips {
		req := distance_request{trip, make(chan distance_response, 1)}
		requests = append(requests, req)
		dist.api.requests <- req
	}

	for _, req := range requests {
		resp := <-req.result
		if resp.error == nil {
			result[req.trip] = resp.distance
			dist.cache.Put(req.trip.Trip, resp.distance)
		}
	}

	return result
}

type DistanceAPI struct {
	api_key  string
	ticker   chan time.Time
	requests chan distance_request
}

type distance_request struct {
	trip   UserTrip
	result chan distance_response
}

type distance_response struct {
	distance uint64
	error    error
}

func NewDistanceAPI(api_key string) DistanceAPI {
	api := DistanceAPI{
		api_key:  api_key,
		ticker:   make(chan time.Time, requests_per_second),
		requests: make(chan distance_request, 50),
	}

	go func() {
		ticker := time.Tick(1e9 / requests_per_second)
		for t := range ticker {
			select {
			case api.ticker <- t:
			default:
			}
		}
	}()

	for i := 0; i < api_workers; i++ {
		go api.worker(i)
	}

	return api
}

func (api DistanceAPI) worker(i int) {
	log.Println("Starting worker", i)
	for {
		<-api.ticker
		request := <-api.requests

		q := url.Values{}
		q.Add("key", api.api_key)
		q.Add("origin", request.trip.From.String())
		q.Add("destination", request.trip.To.String())
		q.Add("mode", "bicycling")

		log.Println("GOOGLE API REQUEST [", i, "]:", api_base, q.Encode())
		resp, err := http.Get(api_base + q.Encode())
		if err != nil {
			request.result <- distance_response{0, err}
			continue
		}

		data, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			request.result <- distance_response{0, err}
			continue
		}

		var response directions_response
		if err := json.Unmarshal(data, &response); err != nil {
			request.result <- distance_response{0, err}
			continue
		}

		distance, err := response.distance()
		if err != nil {
			request.result <- distance_response{0, err}
			continue
		}

		request.result <- distance_response{distance, nil}
	}
}

type directions_response struct {
	Routes []struct {
		Legs []struct {
			Distance struct {
				Text  string
				Value uint64
			}
		}
	}
}

func (response directions_response) distance() (uint64, error) {
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
