package task

import (
	"log"
	"math/rand"
	"sync"
	"time"

	"../cache"
	"../config"
)

/************************************************
Client supports
* tracking hits and misses
* gets back a cache master / hash function?
*************************************************/

type Client struct {
	mu          sync.Mutex
	cachedIDMap map[int]*cache.Cache
	hash        *cache.Hash
	workload    *Workload
	id          int
	startTime   time.Time
}

func (c *Client) BootstrapClient(cachedIDMap map[int]*cache.Cache, hash *cache.Hash, workload *Workload) {
	c.cachedIDMap = cachedIDMap
	c.hash = hash
	c.workload = workload
	c.startTime = time.Now()
}

func Init(id int) *Client {
	c := &Client{}
	c.id = id
	return c
}

func (c *Client) Run() []config.DataType {
	fetched := []config.DataType{}
	for c.workload.HasNextItemGroup() {
		log.Printf("client %v fetching %+v items", c.id, len(c.workload.ItemGroupIndices))
		nextItemGroup := c.workload.GetNextItemGroup()
		fetchedItems := c.fetchItemGroup(nextItemGroup)
		fetched = append(fetched, fetchedItems...)
	}
	return fetched
}

func (c *Client) GetID() int {
	return c.id
}

// ----------------------------------------------- UTILS

func (c *Client) fetchItemGroup(itemGroup []string) []config.DataType {
	items := make([]config.DataType, 0)

	// fetch each item in the group
	for _, itemName := range itemGroup {
		res := c.fetchItem(itemName)
		items = append(items, res)
		// at the end of a web workload pattern, we wait
		if c.workload.workloadName == "web" {
			sleepTime := config.MIN_PATTERN_WAIT + rand.Intn(config.MAX_PATTERN_WAIT-config.MIN_PATTERN_WAIT)
			time.Sleep(time.Duration(sleepTime) * time.Millisecond)
		}
	}
	// make client wait to simulate ML computation
	if c.workload.workloadName == "ml" {
		time.Sleep(config.CLIENT_COMPUTATION_TIME)
	}
	return items
}

func (c *Client) fetchItem(itemName string) config.DataType {
	cacheIds := c.hash.GetCaches(itemName, c.id)
	for _, cacheID := range cacheIds {
		cache := c.cachedIDMap[cacheID]
		item, err := cache.Fetch(itemName)
		if err == nil {
			return item
		}
	}
	// arriving here means all caches are dead
	panic("ALL MY CACHES ARE DEAD. PUSH ME TO THE EDGE")
}
