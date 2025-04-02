package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
)

const (
	defaultPort = 8080
)

var battleServerURL string

func setCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func match(w http.ResponseWriter, r *http.Request) {
	setCORS(w)

	if r.Method == "OPTIONS" {
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"wsUrl": battleServerURL})
}

func main() {
	port := flag.Int("port", defaultPort, "Port to run the lobby server on")
	flag.Parse()

	battleServerURL = fmt.Sprintf("ws://localhost:8081/game")
	addr := fmt.Sprintf(":%d", *port)

	http.HandleFunc("/match", match)

	log.Printf("Lobby server starting on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
