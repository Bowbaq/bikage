package bikage

type NoopCache struct{}

func (c *NoopCache) GetDistance(route Route) (uint64, bool)   { return 0, false }
func (c *NoopCache) PutDistance(route Route, distance uint64) {}

func (c *NoopCache) GetTrip(username, id string) (Trip, bool) { return Trip{}, false }
func (c *NoopCache) GetTrips(username string) Trips           { return Trips{} }
func (c *NoopCache) PutTrip(username string, trip Trip)       {}
