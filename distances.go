package bikage

import "github.com/Bowbaq/distance"

type DistanceCache struct {
	cache Cache
	api   *distance.DirectionsAPI
}

func NewDistanceCache(api_key string, cache Cache) *DistanceCache {
	return &DistanceCache{
		cache: cache,
		api:   distance.NewDirectionsAPI(api_key),
	}
}

func (dist *DistanceCache) Get(trip UserTrip) (uint64, error) {
	distance, ok := dist.cache.Get(trip.Trip)
	if ok {
		return distance, nil
	}

	distance, err := dist.calculate(trip)
	if err != nil {
		return 0, err
	}

	return distance, nil
}

func (dist *DistanceCache) GetAll(trips []UserTrip) map[UserTrip]uint64 {
	result := make(map[UserTrip]uint64)

	var misses []UserTrip
	for _, trip := range trips {
		if distance, ok := dist.cache.Get(trip.Trip); ok {
			result[trip] = distance
		} else {
			misses = append(misses, trip)
		}
	}

	for k, v := range dist.calculate_all(misses) {
		result[k] = v
	}

	return result
}

func (dist *DistanceCache) calculate(trip UserTrip) (uint64, error) {
	distance, err := dist.api.GetDistance(distance.Trip{
		From: trip.From.Coord(),
		To:   trip.To.Coord(),
		Mode: distance.Bicycling,
	})

	if err == nil {
		dist.cache.Put(trip.Trip, distance)
	}

	return distance, err
}

func (dist *DistanceCache) calculate_all(user_trips []UserTrip) map[UserTrip]uint64 {
	trip_for_request := make(map[distance.Trip]UserTrip)
	var requests []distance.Trip
	for _, trip := range user_trips {
		request := distance.Trip{
			From: trip.From.Coord(),
			To:   trip.To.Coord(),
			Mode: distance.Bicycling,
		}
		requests = append(requests, request)
		trip_for_request[request] = trip
	}

	distances := dist.api.GetDistances(requests)

	result := make(map[UserTrip]uint64)
	for request, distance := range distances {
		user_trip := trip_for_request[request]
		result[user_trip] = distance
		dist.cache.Put(user_trip.Trip, distance)
	}

	return result
}
