package config

import (
	"time"
)

const CACHE_SIZE = 50

const PREFETCH_SIZE = 5

//const SEED = time.Now().UnixNano()
const SEED = 1

type CacheType int

const (
	LRU            CacheType = 0
	MarkovPrefetch CacheType = 1
	MarkovEviction CacheType = 2
	MarkovBoth     CacheType = 3
)

type DataType string

// DATA_DEFAULT must be initialized to an empty instance of the above default
const DATA_DEFAULT = ""

const TIME_MULTIPLIER = 100
const DATA_FETCH_TIME = time.Duration(1*TIME_MULTIPLIER) * time.Millisecond

const CLIENT_COMPUTATION_TIME = time.Duration(1*TIME_MULTIPLIER) * time.Millisecond

// latency stuff?

// benchmarking constants -- SMALL
const NFILES_SMALL = 200
const BATCH_SMALL = 16
const ITERS_SMALL = 10
const NCLIENTS_SMALL = 5
const NCACHES_SMALL = 2
const RFACTOR_SMALL = 1
const SYNC_MS_SMALL = 100
