package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type Station struct {
	Id     uint64
	Label  string `json:"stationName"`
	Status int    `json:"statusKey"`
	Coord
}

type Stations map[uint64]Station

type stationResponse struct {
	Ok              bool
	StationBeanList []Station
	LastUpdate      int64
}

func GetStations() (Stations, error) {
	resp, err := http.Get("http://www.citibikenyc.com/stations/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response stationResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	stations := make(Stations)
	for _, s := range response.StationBeanList {
		if s.Status == 1 { // Active station
			stations[s.Id] = s
		}
	}

	return stations, nil
}
