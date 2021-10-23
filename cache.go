package main

import (
	"time"

	"github.com/patrickmn/go-cache"
)

type appCache struct {
	cacheHits int
	cacheMiss int
	cache     *cache.Cache
}

//TODO: retrieve disk cache if exists
func initCache() *appCache {
	var newCache appCache
	newCache.cache = cache.New(10*time.Minute, 15*time.Minute)
	return &newCache
}

func (*appCache) save() error {
	// save cache to disk, at predefined path
	return nil
}

func (c *appCache) get(key string) interface{} {
	cached, found := c.cache.Get(key)
	if found {
		c.cacheHits = c.cacheHits + 1
		return cached
	}
	c.cacheMiss++
	return nil
}

func (c *appCache) add(key string, item interface{}) {
	c.cache.Add(key, item, cache.DefaultExpiration)
}
