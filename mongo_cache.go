package main

import (
	"log"

	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

var (
	collection = "distances"
)

type MongoCache struct {
	session *mgo.Session
}

type CachedDistance struct {
	Id       bson.ObjectId `bson:"_id"`
	From     uint64        `bson:"from"`
	To       uint64        `bson:"to"`
	Distance uint64        `bson:"distance"`
}

func NewMongoCache(mongo_url string) (*MongoCache, error) {
	session, err := mgo.Dial(mongo_url)
	if err != nil {
		return nil, err
	}
	session.SetSafe(&mgo.Safe{})

	return &MongoCache{session}, nil
}

func (c *MongoCache) Get(trip Trip) (uint64, bool) {
	var dist CachedDistance

	query := bson.M{"from": trip.From.Id, "to": trip.To.Id}
	err := c.db().Find(query).One(&dist)

	if err != nil {
		log.Println("MongoCache: GET error -> ", err)
		return 0, false
	}

	return dist.Distance, true
}

func (c *MongoCache) Put(trip Trip, distance uint64) {
	err := c.db().Insert(NewCachedDistance(trip, distance))
	if err != nil {
		log.Println("MongoCache: PUT error -> ", err)
	}
}

func (c *MongoCache) db() *mgo.Collection {
	return c.session.DB("").C(collection)
}

func NewCachedDistance(trip Trip, distance uint64) CachedDistance {
	return CachedDistance{
		Id:       bson.NewObjectId(),
		From:     trip.From.Id,
		To:       trip.To.Id,
		Distance: distance,
	}
}
