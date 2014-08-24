package bikage

import (
	"log"

	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

type MongoCache struct {
	session *mgo.Session
}

func NewMongoCache(mongo_url string) (*MongoCache, error) {
	session, err := mgo.Dial(mongo_url)
	if err != nil {
		return nil, err
	}
	session.SetSafe(&mgo.Safe{})

	username_index := mgo.Index{
		Key:        []string{"username"},
		Unique:     false,
		DropDups:   false,
		Background: true,
		Sparse:     true,
	}
	session.DB("").C("trips").EnsureIndex(username_index)

	trip_id_index := mgo.Index{
		Key:        []string{"username", "trip.id"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}
	session.DB("").C("trips").EnsureIndex(trip_id_index)

	return &MongoCache{session}, nil
}

type CachedRoute struct {
	Id       bson.ObjectId `bson:"_id"`
	From     uint64        `bson:"from"`
	To       uint64        `bson:"to"`
	Distance uint64        `bson:"distance"`
}

func NewCachedRoute(route Route, distance uint64) CachedRoute {
	return CachedRoute{
		Id:       bson.NewObjectId(),
		From:     route.From.Id,
		To:       route.To.Id,
		Distance: distance,
	}
}

func (c *MongoCache) GetDistance(route Route) (uint64, bool) {
	var cached CachedRoute

	s := c.session.Clone()
	defer s.Close()

	query := bson.M{"from": route.From.Id, "to": route.To.Id}
	err := s.DB("").C("routes").Find(query).One(&cached)

	if err != nil {
		log.Println("MongoCache: GET error -> ", err)
		return 0, false
	}

	return cached.Distance, true
}

func (c *MongoCache) PutDistance(route Route, distance uint64) {
	s := c.session.Clone()
	defer s.Close()

	err := s.DB("").C("routes").Insert(NewCachedRoute(route, distance))
	if err != nil {
		log.Println("MongoCache: PUT error -> ", err)
	}
}

type CachedTrip struct {
	MgoId    bson.ObjectId `bson:"_id"`
	Username string
	Trip
}

func NewCachedTrip(username string, trip Trip) CachedTrip {
	return CachedTrip{
		MgoId:    bson.NewObjectId(),
		Username: username,
		Trip:     trip,
	}
}

func (c *MongoCache) GetTrip(username, id string) (Trip, bool) {
	var cached CachedTrip

	s := c.session.Clone()
	defer s.Close()

	query := bson.M{"username": username, "trip.id": id}
	err := s.DB("").C("trips").Find(query).One(&cached)

	if err != nil {
		log.Println("MongoCache: GET error -> ", err)
		return Trip{}, false
	}

	return cached.Trip, true
}

func (c *MongoCache) GetTrips(username string) Trips {
	var cached []CachedTrip

	s := c.session.Clone()
	defer s.Close()

	query := bson.M{"username": username}
	err := s.DB("").C("trips").Find(query).All(&cached)

	trips := make(Trips, 0)

	if err != nil {
		log.Println("MongoCache: GET error -> ", err)
		return trips
	}

	for _, trip := range cached {
		trips = append(trips, trip.Trip)
	}

	return trips
}

func (c *MongoCache) PutTrip(username string, trip Trip) {
	s := c.session.Clone()
	defer s.Close()

	err := s.DB("").C("trips").Insert(NewCachedTrip(username, trip))
	if err != nil && !mgo.IsDup(err) {
		log.Println("MongoCache: PUT error -> ", err)
	}
}
