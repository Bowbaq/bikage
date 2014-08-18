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

func (c *MongoCache) GetTrip(username, id string) (Trip, bool) {
	return Trip{}, false
}

func (c *MongoCache) PutTrip(trip Trip) {}
