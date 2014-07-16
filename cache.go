package main

type Cache interface {
	Get(trip Trip) (uint64, bool)
	Put(trip Trip, distance uint64)
}
