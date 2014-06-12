package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

func GetStations() (map[uint64]Station, error) {
	resp, err := http.Get("http://www.citibikenyc.com/stations/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response StationResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	stations := make(map[uint64]Station)
	for _, s := range response.StationBeanList {
		if s.Status == 1 { // Active station
			stations[s.Id] = s
		}
	}

	return stations, nil
}
