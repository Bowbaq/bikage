package bikage

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
)

type Bikage struct {
	distance_cache *DistanceCache
	stations       Stations
}

func NewBikage(google_api_key string, cache Cache) (*Bikage, error) {
	stations, err := GetStations()
	if err != nil {
		return nil, errors.New("Bikage STATIONS GET error -> " + err.Error())
	}

	bikage := Bikage{
		distance_cache: NewDistanceCache(google_api_key, cache),
		stations:       stations,
	}

	return &bikage, nil
}

func (bk *Bikage) GetTrips(username, password string) ([]UserTrip, error) {
	return get_trips(username, password, bk.stations)
}

func (bk *Bikage) ComputeStats(trips []UserTrip) *Stats {
	distances := bk.distance_cache.GetAll(trips)

	stats := NewStats()
	for _, trip := range trips {
		dist, ok := distances[trip]
		if !ok {
			log.Println("Bikage GET TRIP DISTANCE error, trip:", trip)
			continue
		}

		day := trip.StartedAt.Truncate(24 * time.Hour)
		if day_total, ok := stats.DailyTotal[day]; ok {
			stats.DailyTotal[day] = day_total + dist
		} else {
			stats.DailyTotal[day] = dist
		}

		stats.Total += dist
	}

	return stats
}

type Stats struct {
	Total      uint64
	DailyTotal map[time.Time]uint64
}

func NewStats() *Stats {
	return &Stats{DailyTotal: make(map[time.Time]uint64)}
}

func (s *Stats) TotalKm() float64 {
	return km_dist(s.Total)
}

func (s *Stats) TotalMi() float64 {
	return mi_dist(s.Total)
}

func (s Stats) String() string {
	summaries := make([]string, 0)
	for day, dist := range s.DailyTotal {
		summary := fmt.Sprintf("  %s %.1f km (%.1f mi)", day.Format("01/02/2006"), km_dist(dist), mi_dist(dist))
		summaries = append(summaries, summary)
	}
	sort.Strings(summaries)

	return fmt.Sprintf("Total:\n  %.1f km (%.1f mi)\nDetails:\n%s", s.TotalKm(), s.TotalMi(), strings.Join(summaries, "\n"))
}

func km_dist(dist uint64) float64 {
	return float64(dist) / 1000
}

func mi_dist(dist uint64) float64 {
	return float64(dist) / 1000 * 0.621371192
}
