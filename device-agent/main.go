package main

import (
	"log"
	"device-agent/service"
)

func main() {
	log.Println("Starting Device Agent")
	service.Run()
}
