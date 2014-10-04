package bikage

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/Bowbaq/distance"
)

type Bikage struct {
	RouteAPI RouteAPI
	TripAPI  TripAPI
}

const DayFormat = "01/02/2006 EST"

func NewBikage(google_api_key string, mongo_url string) (*Bikage, error) {
	stations, err := GetStations()
	if err != nil {
		return nil, errors.New("Bikage STATIONS GET error -> " + err.Error())
	}

	var cache Cache
	cache, err = NewMongoCache(mongo_url)
	if err != nil {
		cache = NewJsonCache()
	}

	directions_api := distance.NewDirectionsAPI(google_api_key)

	bikage := Bikage{
		RouteAPI: NewRouteAPI(directions_api).WithCache(cache),
		TripAPI:  NewTripAPI(stations).WithCache(cache),
	}

	return &bikage, nil
}

func (bk *Bikage) GetTrips(username, password string) (Trips, error) {
	return bk.TripAPI.GetTrips(username, password)
}

func (bk *Bikage) GetCachedTrips(username string) Trips {
	return bk.TripAPI.GetCachedTrips(username)
}

func (bk *Bikage) ComputeStats(trips Trips) *Stats {
	distances := bk.RouteAPI.GetAll(trips)

	stats := NewStats()
	for _, trip := range trips {
		dist, ok := distances[trip]
		if !ok {
			log.Println("Bikage COULDN'T COMPUTE DISTANCE FOR ROUTE error, trip:", trip)
			continue
		}

		day := trip.StartedAt.Format(DayFormat)
		if day_distance_total, ok := stats.DailyDistanceTotal[day]; ok {
			stats.DailyDistanceTotal[day] = day_distance_total + dist
		} else {
			stats.DailyDistanceTotal[day] = dist
		}

		if day_speed_total, ok := stats.DailySpeedTotal[day]; ok {
			stats.DailySpeedTotal[day] = day_speed_total + float64(dist)/trip.Duration().Hours()
		} else {
			stats.DailySpeedTotal[day] = float64(dist) / trip.Duration().Hours()
		}

		stats.Total += dist

		stats.TotalTime += trip.Duration()
	}

	stats.AvgSpeed = stats.TotalKm() / stats.TotalTime.Hours()

	return stats
}

type Stats struct {
	Total              uint64
	TotalTime          time.Duration
	DailyDistanceTotal map[string]uint64
	DailySpeedTotal    map[string]float64
	AvgSpeed           float64
}

func NewStats() *Stats {
	return &Stats{DailyDistanceTotal: make(map[string]uint64), DailySpeedTotal: make(map[string]float64)}
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
	return float64(dist) / 1609.34
}

func (s Stats) String() string {
	summaries := make([]string, 0)
	for day, dist := range s.DailyDistanceTotal {
		summary := fmt.Sprintf("  %s %.1f km (%.1f mi)", day, km_dist(dist), mi_dist(dist))
		summaries = append(summaries, summary)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(summaries)))

	return fmt.Sprintf("Total:\n  %.1f km (%.1f mi)\nDetails:\n%s", s.TotalKm(), s.TotalMi(), strings.Join(summaries, "\n"))
}
