package main

import (
	"flowaggs/internal"
	"log"
)

func main() {
	err := internal.InitRESTServer()
	if err != nil {
		log.Fatalf("Could not start REST server: %s. Exiting...", err.Error())
	}
}
