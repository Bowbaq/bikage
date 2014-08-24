package bikage

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
)

type Bikage struct {
	route_cache *RouteCache
	trip_cache  *TripCache
}

const DayFormat = "01/02/2006"

func NewBikage(google_api_key string, cache Cache) (*Bikage, error) {
	stations, err := GetStations()
	if err != nil {
		return nil, errors.New("Bikage STATIONS GET error -> " + err.Error())
	}

	bikage := Bikage{
		route_cache: NewRouteCache(cache, google_api_key),
		trip_cache:  NewTripCache(cache, stations),
	}

	return &bikage, nil
}

func (bk *Bikage) GetTrips(username, password string) (Trips, error) {
	return bk.trip_cache.GetTrips(username, password)
}

func (bk *Bikage) ComputeStats(trips Trips) *Stats {
	distances := bk.route_cache.GetAll(trips)

	stats := NewStats()
	for _, trip := range trips {
		dist, ok := distances[trip]
		if !ok {
			log.Println("Bikage COULDN'T COMPUTE DISTANCE FOR ROUTE error, trip:", trip)
			continue
		}

		day := trip.StartedAt.Format(DayFormat)
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
	DailyTotal map[string]uint64
}

func NewStats() *Stats {
	return &Stats{DailyTotal: make(map[string]uint64)}
}

func (s *Stats) TotalKm() float64 {
	return km_dist(s.Total)
}

func km_dist(dist uint64) float64 {
	return float64(dist) / 1000
}

func (s *Stats) TotalMi() float64 {
	return mi_dist(s.Total)
}

func mi_dist(dist uint64) float64 {
	return float64(dist) / 1000 * 0.621371192
}

func (s Stats) String() string {
	summaries := make([]string, 0)
	for day, dist := range s.DailyTotal {
		summary := fmt.Sprintf("  %s %.1f km (%.1f mi)", day, km_dist(dist), mi_dist(dist))
		summaries = append(summaries, summary)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(summaries)))

	return fmt.Sprintf("Total:\n  %.1f km (%.1f mi)\nDetails:\n%s", s.TotalKm(), s.TotalMi(), strings.Join(summaries, "\n"))
}
