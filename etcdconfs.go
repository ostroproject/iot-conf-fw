package main

import (
	"flag"
	"log"
	"strings"
	"ostro/etcd/confs"
)

func main() {
	var servers, prefix, cache string

        flag.StringVar(&servers, "servers", "http://localhost:2379", "comma separated list of URLS")
	flag.StringVar(&prefix, "prefix", "/", "etcd key prefix")
	flag.StringVar(&cache, "cache", "/var/cache/confs", "etcd key hierarchy will go under this directory")

	flag.Parse()

	serverList := strings.Split(strings.Replace(servers, " ", "", -1), ",")
	
	cfs := confs.NewConFS(serverList, prefix, cache)

	changes, err := cfs.TraverseEtcdTree()
	if err != nil {
		log.Fatal("failed to initialize.\n")
	}
	
	if err := cfs.SyncFiles(changes, false); err != nil {
		log.Fatalf("failed to update cache: %v\n", err)
	}
	
	change_chan, err_chan := cfs.PollForChanges()
	
	for {
		select {
		case changes := <-change_chan:
			cfs.SyncFiles(changes, true)
		case err := <-err_chan:
			log.Fatalf("polling failed: %v\n", err)
		}
	}
}
