package main

import (
	"device-agent-linux/service"
	"log"
)

func main() {
	log.Println("Starting Device Agent for Linux")
	service.Run()
}
