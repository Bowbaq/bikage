package main

import (
	"time"
)

type Coord struct {
	Lat float64 `json:"latitude"`
	Lng float64 `json:"longitude"`
}

type Station struct {
	Id     uint64
	Label  string `json:"stationName"`
	Status int    `json:"statusKey"`
	Coord
}

type Trip struct {
	From Station
	To   Station
}

type UserTrip struct {
	Trip
	StartedAt time.Time
	EndedAt   time.Time
}

type Distances struct {
	cache   map[string]uint64
	api_key string
}

type StationResponse struct {
	Ok              bool
	StationBeanList []Station
	LastUpdate      int64
}

type DirectionsResponse struct {
	Routes []struct {
		Legs []struct {
			Distance struct {
				Text  string
				Value uint64
			}
		}
	}
}
