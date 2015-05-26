package bikage

import (
	"encoding/json"
	"net/http"

	"github.com/Bowbaq/distance"
)

const stations_endpoint = "http://www.citibikenyc.com/stations/json"

type Stations map[string]Station

type Station struct {
	Id     uint64
	Label  string  `json:"stationName"`
	Status int     `json:"statusKey"`
	Lat    float64 `json:"latitude"`
	Lng    float64 `json:"longitude"`
}

func (s Station) String() string {
	return s.Label
}

func GetStations() (Stations, error) {
	resp, err := http.Get(stations_endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response struct {
		StationBeanList []Station
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	stations := make(Stations)
	for _, s := range response.StationBeanList {
		if s.Status == 1 { // Active station
			stations[s.Label] = s
		}
	}

	return stations, nil
}

func (s Station) Coord() distance.Coord {
	return distance.Coord{
		Lat: s.Lat,
		Lng: s.Lng,
	}
}
