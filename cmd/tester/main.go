package main

import (
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {

	panicOnError(fmt.Errorf("not implemented"))
}

func panicOnError(err error) {
	if err != nil {
		log.Panic(err)
	}
}
