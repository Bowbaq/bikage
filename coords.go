package main

import "fmt"

type Coord struct {
	Lat float64 `json:"latitude"`
	Lng float64 `json:"longitude"`
}

func (coord Coord) String() string {
	return fmt.Sprintf("%.8f,%.8f", coord.Lat, coord.Lng)
}
