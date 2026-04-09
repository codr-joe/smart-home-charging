package main

import (
	"encoding/json"
	"flag"
	"log"
	"math"
	"net/http"
	"time"
)

type p1Response struct {
	ActivePowerW float64 `json:"active_power_w"`
	ActiveTariff int     `json:"active_tariff"`
}

func main() {
	addr := flag.String("addr", ":8090", "address to listen on")
	flag.Parse()
	start := time.Now()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/data", func(w http.ResponseWriter, r *http.Request) {
		elapsed := time.Since(start).Seconds()
		solar := 3000 * math.Max(0, math.Sin(math.Pi*elapsed/43200))
		resp := p1Response{
			ActivePowerW: 400.0 - solar + 50*math.Sin(elapsed/7),
			ActiveTariff: 1,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("encode error: %v", err)
		}
	})
	log.Printf("p1mock: listening on %s", *addr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatalf("p1mock: %v", err)
	}
}
