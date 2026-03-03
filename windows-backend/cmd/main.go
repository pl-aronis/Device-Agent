package main

import (
	"log"
	"net/http"
	"windows-backend/internal/api"
)

func main() {
	router := api.NewRouter()

	log.Println("Backend running on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
