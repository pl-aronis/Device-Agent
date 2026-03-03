package api

import (
	"net/http"
	"windows-backend/internal/service"
	"windows-backend/internal/store"
)

func NewRouter() http.Handler {

	store := store.NewJSONStore("data/devices.json")
	service := service.NewDeviceService(store)

	h := NewHandler(service)

	mux := http.NewServeMux()

	mux.HandleFunc("/register", h.Register)
	mux.HandleFunc("/re-authenticate", h.ReAuth)
	mux.HandleFunc("/heartbeat", h.Heartbeat)
	mux.HandleFunc("/lock-success", h.LockSuccess)
	mux.HandleFunc("/lock-failure", h.LockFailure)
	mux.HandleFunc("/admin/status", h.AdminStatus)
	mux.HandleFunc("/admin/set", h.AdminSet)

	return withCORS(mux)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
