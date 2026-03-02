package app

import (
	"device-agent-windows/internal/enforcement"
	"log"
)

func Run() {
	err := run()
	if err != nil {
		log.Println(err)
	}
}

func run() error {

	enforcement.EnforceDeviceLock()

	return nil
}
