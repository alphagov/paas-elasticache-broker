package main

import (
	"fmt"
	"github.com/alphagov/paas-elasticache-broker/providers/redis"
	"os"
)

func main(){
	if len(os.Args) < 2 {
		fmt.Println("Must provide a service GUID as the first argument")
		os.Exit(1)
		return
	}

	id := os.Args[1]
	hash := redis.GenerateReplicationGroupName(id)

	fmt.Println(fmt.Sprintf("GUID: %s", id))
	fmt.Println(fmt.Sprintf("Hash: %s", hash))
}
