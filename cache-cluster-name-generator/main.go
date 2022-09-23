package main

import (
	"fmt"
	"os"

	"github.com/alphagov/paas-elasticache-broker/providers/redis"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Must provide a service GUID as the first argument")
		os.Exit(1)
		return
	}

	id := os.Args[1]
	hash := redis.GenerateReplicationGroupName(id)

	fmt.Printf("GUID: %s\n", id)
	fmt.Printf("Hash: %s\n", hash)
}
