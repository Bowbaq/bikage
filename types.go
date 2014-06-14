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

type Cache interface {
	Get(trip Trip) (uint64, bool)
	Put(trip Trip, distance uint64)
}

type DistanceCache struct {
	cache   Cache
	api_key string
}
