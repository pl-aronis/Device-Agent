package device

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type Location struct {
	Latitude  float64
	Longitude float64
}

type ipInfo struct {
	Loc string `json:"loc"`
}

func GetLocation() Location {
	resp, err := http.Get("https://ipinfo.io/json")
	if err != nil {
		return Location{}
	}
	defer resp.Body.Close()

	var data ipInfo
	json.NewDecoder(resp.Body).Decode(&data)

	parts := strings.Split(data.Loc, ",")
	if len(parts) != 2 {
		return Location{}
	}

	lat, _ := strconv.ParseFloat(parts[0], 64)
	lon, _ := strconv.ParseFloat(parts[1], 64)

	return Location{Latitude: lat, Longitude: lon}
}
