package cache

import (
	"sync"
	"errors"
	"../heap"
	"../markov"
	"../datastore"
	"../config"
	// "../utils"
    "log"
)

/********************************
Cache supports the following external API to users

c.Init(cacheSize int, cacheType CacheType, data *datastore.DataStore)
	Initializes a cache with eviction policy and prefetch defined by cache type
	Copies underlying datastore
c.Report() (hits, misses)
    Get a report of the hits and misses  TODO: Do we want a version number or
    timestamp mechanism of any form here?
c.Fetch(name string) (config.DataType, error)

*********************************/
type Cache struct {
	mu          sync.Mutex          // Lock to protect shared access to cache
    id          int
	misses		int64
	hits		int64
    cache	    map[string]config.DataType
	heap		*heap.MinHeapInt64
	timestamp	int64 // for controlling LRU heap
	cacheSize	int
	chain		*markov.MarkovChain
	cacheType	config.CacheType
	data		*datastore.DataStore
    alive       bool
}

// client -> cache (Request a file)
type RequestFileArgs struct {
	Filename 	string
	ID 			int
}

type RequestFileReply struct {
	File	config.DataType
}

// copies underlying datastore
func (c *Cache) Init(id int, cacheSize int, cacheType config.CacheType, data *datastore.DataStore) {
	c.cacheType = cacheType
    c.id = id
	c.misses = 0
	c.hits = 0
    c.cacheSize = cacheSize
	c.cache = make(map[string]config.DataType)
	c.timestamp = 0
	c.data = data.Copy()
    c.alive = true

	if cacheType != config.MarkovEviction {
		// only LRU caches should use heap
		c.heap = heap.MakeMinHeapInt64()
	}
	if cacheType != config.LRU {
		// all other caches need a MarkovChain
		c.chain = markov.MakeMarkovChain()
	}
}

func (c* Cache) Killed() bool {
    c.mu.Lock()
    defer c.mu.Unlock()
    return !c.alive
}

func (c* Cache) killed() bool {
    return !c.alive
}

func (c* Cache) Kill() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.alive = false
}

func (c* Cache) Revive() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.alive = true
}

func (c *Cache) GetId() int {
    return c.id
}

func (c *Cache) Fetch(name string, id int) (config.DataType, error) {
	// log.Printf("Entering Fetch %v", name)
	// defer utils.DPrintf("Leaving Fetch %v", name)
	c.mu.Lock()
	defer c.mu.Unlock()
	var err error
    if c.killed() {
        err = errors.New("Error: Cache Is Dead")
        return config.DATA_DEFAULT, err
    }

	
	// log.Printf("%v, %v, %v", len(c.cache),  c.heap.Size, c.cacheSize)
	// log.Printf("%v", c.cache)
	// if len(c.cache) != c.heap.Size || (len(c.cache) > c.cacheSize) || (c.heap.Size > c.cacheSize) {
	// 	log.Fatalf("FAILED: %v, != %v", len(c.cache), c.heap.Size)
	// }
	file, ok := c.cache[name]
	

	if c.cacheType != config.LRU {
		// all other caches need a MarkovChain
		c.chain.Access(name, id)
	}

	if ok {
		c.hits++
		err = nil

		// TODO: THIS IS BAAD PRACTICE BUT WILL SUFFICE FOR NOW
		if c.cacheType != config.MarkovEviction {
			// only LRU caches should use heap
			c.heap.ChangeKey(name, c.timestamp)
		}
	} else {
		file = c.AddToCache(name)
		c.misses++
	}
	
	if c.timestamp % config.PREFETCH_SIZE == 0 {
		go c.Prefetch(name)
	}
	return file, err
}


func (c *Cache) Report() (int64, int64, int64) {
    c.mu.Lock()
    defer c.mu.Unlock()
	return c.hits, c.misses, c.data.CountCalls()
}

// TODO: REPLACEMENT POLICY FOR MARKOV CHAIN
// assumes mu is Locked
func (c *Cache) replace(name string, file config.DataType) {
	// c.mu.Lock()
	// defer c.mu.Unlock()
	c.cache[name] = file
	c.heap.Insert(name, c.timestamp)
	c.timestamp++
	if c.heap.Size > c.cacheSize {
		// must evict
		evict := c.heap.ExtractMin()
		delete(c.cache, evict)
	}
}

func (c *Cache) Prefetch(filename string) {
	if c.cacheType == config.MarkovPrefetch || c.cacheType == config.MarkovBoth {
		files := c.chain.Predict(filename, config.PREFETCH_SIZE)
		c.mu.Lock()
		for _, file :=  range files {
			// log.Printf("Predicted %v after %v",  file, filename)
			c.AddToCache(file)
		}
		c.mu.Unlock()
	}
}

func (c *Cache) GetState(prevChain *markov.MarkovChain) *markov.MarkovChain {
    // TODO: Need some if statements around cache type here
    c.mu.Lock()
    defer c.mu.Unlock()
    return markov.ChainSub(c.chain, prevChain)
}

func (c *Cache) UpdateState(chain *markov.MarkovChain) {
    // TODO: Need some if statements around cache type here
    c.mu.Lock()
    defer c.mu.Unlock()
    c.chain = chain.Copy()
}

func (c *Cache) AddToCache(filename string) config.DataType {
	file, ok := c.cache[filename]
	
	if !ok {
		// log.Printf("getting file %v", filename)
		file, ok = c.data.Get(filename)
		if !ok {
			log.Fatalf("Failed to fetch file %v", filename)
		}
		
		c.replace(filename, file) // handles insertion into heap
	}
	return file
}


func (c *Cache) FetchRPC(args *RequestFileArgs, reply *RequestFileReply) error {
	var err error
	reply.File, err = c.Fetch(args.Filename, args.ID)
	return err
}
