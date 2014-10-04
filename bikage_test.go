package bikage_test

import (
	"time"

	. "github.com/Bowbaq/bikage"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("bikage", func() {
	Describe("ComputeStats()", func() {
		route_api := test_route_api{}
		trip_api := test_trip_api{}

		bk := &Bikage{
			RouteAPI: &route_api,
			TripAPI:  &trip_api,
		}

		var stats *Stats

		Context("without any trips", func() {
			stats = bk.ComputeStats(Trips{})

			It("returns 0 total miles biked", func() {
				Expect(stats.Total).To(BeZero())
			})

			It("has no dailt totals", func() {
				Expect(stats.DailyDistanceTotal).To(HaveLen(0))
			})
		})

		Context("with trips data", func() {
			today := time.Now()
			yesterday := today.AddDate(0, 0, -1)

			trip1 := Trip{Id: "1", StartedAt: yesterday}
			trip2 := Trip{Id: "2", StartedAt: today}
			trip3 := Trip{Id: "3", StartedAt: today}
			trips := Trips{trip1, trip2, trip3}

			BeforeEach(func() {
				route_api.get_all = func(trips Trips) map[Trip]uint64 {
					return map[Trip]uint64{
						trip1: 1000,
						trip2: 3000,
						trip3: 2000,
					}
				}

				stats = bk.ComputeStats(trips)
			})

			Describe("stats.Total", func() {
				It("should contain the sum of the distance for each trip in meters", func() {
					Expect(stats.Total).To(BeNumerically("==", 6000))
				})
			})

			Describe("stats.DailyDistanceTotal", func() {
				It("should have an entry with the distance in meters for each day", func() {
					Expect(stats.DailyDistanceTotal).To(HaveKeyWithValue(yesterday.Format(DayFormat), BeNumerically("==", 1000)))
					Expect(stats.DailyDistanceTotal).To(HaveKeyWithValue(today.Format(DayFormat), BeNumerically("==", 5000)))
				})
			})

			AfterEach(func() {
				route_api.get_all = nil
			})
		})
	})

	Describe("Stats", func() {
		stats := NewStats()
		stats.Total = 5000

		Describe("stats.TotalKm()", func() {
			It("returns the total distance in kilometers", func() {
				Expect(stats.TotalKm()).To(BeNumerically("==", 5))
			})
		})

		Describe("stats.TotalMi()", func() {
			It("returns the total distance in miles", func() {
				Expect(stats.TotalMi()).To(BeNumerically("==", float64(5000)/1609.34))
			})
		})
	})
})

type test_route_api struct {
	get_all func(trips Trips) map[Trip]uint64
}

func (tra *test_route_api) WithCache(cache DistanceCache) RouteAPI { return tra }
func (tra *test_route_api) Get(trip Trip) (uint64, error)          { return 0, nil }
func (tra *test_route_api) GetAll(trips Trips) map[Trip]uint64 {
	if tra.get_all != nil {
		return tra.get_all(trips)
	}

	return make(map[Trip]uint64)
}

type test_trip_api struct{}

func (tta *test_trip_api) WithCache(cache TripCache) TripAPI                 { return tta }
func (tta *test_trip_api) GetTrips(username, password string) (Trips, error) { return Trips{}, nil }
func (tta *test_trip_api) GetCachedTrips(username string) Trips              { return Trips{} }
