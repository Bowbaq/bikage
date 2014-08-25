package bikage

type DistanceCache interface {
	GetDistance(route Route) (uint64, bool)
	PutDistance(route Route, distance uint64)
}

type TripCache interface {
	GetTrip(username, id string) (Trip, bool)
	GetTrips(username string) Trips
	PutTrip(username string, trip Trip)
}

type Cache interface {
	DistanceCache
	TripCache
}
