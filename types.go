package main

import (
	"sync"
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

type Stations map[uint64]Station

type Trip struct {
	From Station
	To   Station
}

type UserTrip struct {
	Trip
	StartedAt time.Time
	EndedAt   time.Time
}

type DistanceCache struct {
	cache   map[string]uint64
	api_key string
	sync.RWMutex
}
