package task

import (
	"fmt"
	"log"
	"strconv"
	"testing"
	"../datastore"
	"../config"
	"../utils"
	"../cache"
)

func TestClientSimpleWorkload(t *testing.T) {
	fmt.Println("TestClientSimpleWorkload ...")

	numFiles := 100
	numClients := 1
	numCaches := 1
	syncCachesEveryMS := 100
	replicationFactor := 1

	// make datastore
	data := datastore.MakeDataStore()
	for i := 0; i < numFiles; i++ {
		filename := "fake_" + strconv.Itoa(i) + ".txt"
		data.Make(filename)
	}

	// make basic workload
	fileNames := data.GetFileNames()
	itemGroupIndices := make([][]int, len(fileNames))
	for i := 0; i < len(fileNames); i++ {
		itemGroupIndices[i] = []int{i}
	}
	w := &Workload{ItemNames: fileNames, ItemGroupIndices: itemGroupIndices}

	// make clients backbone
	clients := make([]*Client, numClients)
	for i := 0; i < numClients; i++ {
		clients[i] = Init(i)
	}
	clientIds := make([]int, len(clients))
	for i := 0; i < len(clients); i++ {
		clientIds[i] = clients[i].GetID()
	}
	cachedIDMap, hash := StartTask(clientIds, config.LRU, config.CACHE_SIZE, numCaches, replicationFactor, data, syncCachesEveryMS)
	for i := 0; i < numClients; i++ {
		clients[i].BootstrapClient(cachedIDMap, *hash, w)
	}

	for _, c := range clients {
		log.Println(c.Run())
	}
}


func TestHashEndToEnd(t *testing.T) {
	fmt.Printf("TestHashmakeFileGroups ...\n")
	failed := false

	// case 0
	numCaches := 7
	filenames := []string{"a", "b", "c", "d", "e",
		"f", "g", "h", "i", "j",
		"k", "l", "m"}
	replication := 2
	numClients := 4
	clients := make([]*Client, numClients)
	ids := make([]int, numClients)
	for i := 0; i < numClients; i++ {
		clients[i] = Init(i)
		ids[i] = i
	}
	
	hash := cache.MakeHash(numCaches, filenames, len(filenames), replication, ids)

	file := "a"
	first := hash.GetCaches(file, 0)
	second := hash.GetCaches(file, 1)
	third := hash.GetCaches(file, 2)
	fourth := hash.GetCaches(file, 3)

	if !utils.IntArraySetsEqual(first, second) || !utils.IntArraySetsEqual(first, third) || !utils.IntArraySetsEqual(first, fourth) {
		failed = true
		t.Errorf("Expected same cache id sets for each client id, but got: %v, %v, %v, and %v for file %v", first, second, third, fourth, file)
	}

	if len(first) < replication || len(first) > replication+1 {
		failed = true
		t.Errorf("Got bad replication group size: %v when numcaches is %v and replication is %v", len(first), numCaches, replication)
	}

	file = "b"
	first = hash.GetCaches(file, 0)
	second = hash.GetCaches(file, 1)
	third = hash.GetCaches(file, 2)
	fourth = hash.GetCaches(file, 3)

	if !utils.IntArraySetsEqual(first, second) || !utils.IntArraySetsEqual(first, third) || !utils.IntArraySetsEqual(first, fourth) {
		failed = true
		t.Errorf("Expected same cache id sets for each client id, but got: %v, %v, %v, and %v for file %v", first, second, third, fourth, file)
	}

	if len(first) < replication || len(first) > replication+1 {
		failed = true
		t.Errorf("Got bad replication group size: %v when numcaches is %v and replication is %v", len(first), numCaches, replication)
	}

	if failed {
		fmt.Printf("\t... FAILED\n")
	} else {
		fmt.Printf("\t... PASSED\n")
	}
}