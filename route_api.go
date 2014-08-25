package bikage

import "github.com/Bowbaq/distance"

type DirectionsAPI interface {
	GetDistance(trip distance.Trip) (uint64, error)
	GetDistances(trips []distance.Trip) map[distance.Trip]uint64
}

type RouteAPI interface {
	WithCache(cache DistanceCache) RouteAPI

	Get(trip Trip) (uint64, error)
	GetAll(trips Trips) map[Trip]uint64
}

type route_api struct {
	cache DistanceCache
	api   DirectionsAPI
}

func NewRouteAPI(directions_api DirectionsAPI) RouteAPI {
	return &route_api{
		cache: new(NoopCache),
		api:   directions_api,
	}
}

func (ra *route_api) WithCache(cache DistanceCache) RouteAPI {
	ra.cache = cache
	return ra
}

func (ra *route_api) Get(trip Trip) (uint64, error) {
	distance, ok := ra.cache.GetDistance(trip.Route)
	if ok {
		return distance, nil
	}

	distance, err := ra.calculate(trip.Route)
	if err != nil {
		return 0, err
	}

	return distance, nil
}

func (ra *route_api) GetAll(trips Trips) map[Trip]uint64 {
	result := make(map[Trip]uint64)

	var misses Trips
	for _, trip := range trips {
		if distance, ok := ra.cache.GetDistance(trip.Route); ok {
			result[trip] = distance
		} else {
			misses = append(misses, trip)
		}
	}

	for k, v := range ra.calculate_all(misses) {
		result[k] = v
	}

	return result
}

func (ra *route_api) calculate(route Route) (uint64, error) {
	distance, err := ra.api.GetDistance(distance.Trip{
		From: route.From.Coord(),
		To:   route.To.Coord(),
		Mode: distance.Bicycling,
	})

	if err == nil {
		ra.cache.PutDistance(route, distance)
	}

	return distance, err
}

func (ra *route_api) calculate_all(trips Trips) map[Trip]uint64 {
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

	distances := ra.api.GetDistances(requests)

	result := make(map[Trip]uint64)
	for request, distance := range distances {
		trip := trip_for_request[request]
		result[trip] = distance
		ra.cache.PutDistance(trip.Route, distance)
	}

	return result
}
