package bikage_test

import (
	"time"

	. "github.com/Bowbaq/bikage"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Bikage", func() {
	Describe("computing stats", func() {
		route_api := test_route_api{}
		trip_api := test_trip_api{}

		bk := &Bikage{
			RouteAPI: &route_api,
			TripAPI:  &trip_api,
		}

		Context("with no trips", func() {
			trips := Trips{}

			It("should have a total distance of zero", func() {
				stats := bk.ComputeStats(trips)
				Expect(stats.Total).To(BeZero())
			})

			It("should have an empty map of daily totals", func() {
				stats := bk.ComputeStats(trips)
				Expect(stats.DailyTotal).To(HaveLen(0))
			})
		})

		Context("with some trips", func() {
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
			})

			Context("stats.Total", func() {
				It("should be the sum of the distances", func() {
					stats := bk.ComputeStats(trips)
					Expect(stats.Total).To(BeNumerically("==", 6000))
				})
			})

			Context("stats.TotalKm()", func() {
				It("should be the sum of the distances / 1000", func() {
					stats := bk.ComputeStats(trips)
					Expect(stats.TotalKm()).To(BeNumerically("==", 6))
				})
			})

			Context("stats.TotalMi()", func() {
				It("should be the sum of the distances / 1609.34", func() {
					stats := bk.ComputeStats(trips)
					Expect(stats.TotalMi()).To(BeNumerically("==", float64(6000)/1609.34))
				})
			})

			Context("stats.DailyTotal", func() {
				It("should have an entry with the sum for each day", func() {
					stats := bk.ComputeStats(trips)
					Expect(stats.DailyTotal).To(HaveKeyWithValue(yesterday.Format(DayFormat), BeNumerically("==", 1000)))
					Expect(stats.DailyTotal).To(HaveKeyWithValue(today.Format(DayFormat), BeNumerically("==", 5000)))
				})
			})

			AfterEach(func() {
				route_api.get_all = nil
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
