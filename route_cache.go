package bikage

import "github.com/Bowbaq/distance"

type RouteCache struct {
	cache Cache
	api   *distance.DirectionsAPI
}

func NewRouteCache(cache Cache, api_key string) *RouteCache {
	return &RouteCache{
		cache: cache,
		api:   distance.NewDirectionsAPI(api_key),
	}
}

func (rc *RouteCache) Get(trip Trip) (uint64, error) {
	distance, ok := rc.cache.GetDistance(trip.Route)
	if ok {
		return distance, nil
	}

	distance, err := rc.calculate(trip.Route)
	if err != nil {
		return 0, err
	}

	return distance, nil
}

func (rc *RouteCache) GetAll(trips Trips) map[Trip]uint64 {
	result := make(map[Trip]uint64)

	var misses Trips
	for _, trip := range trips {
		if distance, ok := rc.cache.GetDistance(trip.Route); ok {
			result[trip] = distance
		} else {
			misses = append(misses, trip)
		}
	}

	for k, v := range rc.calculate_all(misses) {
		result[k] = v
	}

	return result
}

func (rc *RouteCache) calculate(route Route) (uint64, error) {
	distance, err := rc.api.GetDistance(distance.Trip{
		From: route.From.Coord(),
		To:   route.To.Coord(),
		Mode: distance.Bicycling,
	})

	if err == nil {
		rc.cache.PutDistance(route, distance)
	}

	return distance, err
}

func (rc *RouteCache) calculate_all(trips Trips) map[Trip]uint64 {
	trip_for_request := make(map[distance.Trip]Trip)
	var requests []distance.Trip
	for _, trip := range trips {
		request := distance.Trip{
			From: trip.Route.From.Coord(),
			To:   trip.Route.To.Coord(),
			Mode: distance.Bicycling,
		}
		requests = append(requests, request)
		trip_for_request[request] = trip
	}

	distances := rc.api.GetDistances(requests)

	result := make(map[Trip]uint64)
	for request, distance := range distances {
		trip := trip_for_request[request]
		result[trip] = distance
		rc.cache.PutDistance(trip.Route, distance)
	}

	return result
}
