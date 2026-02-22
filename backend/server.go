package main

import (
	"log"
	"net/http"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	store := NewStorage("data/devices.json")

	mux := http.NewServeMux()
	mux.HandleFunc("/api/register", registerHandler(store))
	mux.HandleFunc("/api/heartbeat", heartbeatHandler(store))
	mux.HandleFunc("/api/recovery-key", recoveryKeyHandler(store))
	mux.HandleFunc("/admin/set", adminSetHandler(store))
	mux.HandleFunc("/admin/status", adminStatusHandler(store))
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		log.Println("[BACKEND] Ping received")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	})

	handler := corsMiddleware(mux)

	log.Println("Backend listening on :8080")
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", handler))
}
