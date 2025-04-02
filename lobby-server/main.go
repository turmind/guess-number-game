package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	defaultPort  = 8080
	matchTimeout = 180 * time.Second
)

var (
	battleServerURL string
	waitingPlayer   chan struct{} // Channel to signal waiting player
	matchMutex      sync.Mutex
)

type MatchResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	WsUrl   string `json:"wsUrl,omitempty"`
}

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
	w.Header().Set("Transfer-Encoding", "chunked")
	w.(http.Flusher).Flush()

	matchMutex.Lock()
	if waitingPlayer == nil {
		// First player - create waiting channel
		waitingPlayer = make(chan struct{})
		matchMutex.Unlock()

		// Send waiting response and ensure it's flushed
		if err := json.NewEncoder(w).Encode(MatchResponse{
			Status:  "waiting",
			Message: "Waiting for opponent...",
		}); err != nil {
			log.Printf("Error sending waiting response: %v", err)
			return
		}
		w.(http.Flusher).Flush()

		// Wait for second player or timeout
		select {
		case <-waitingPlayer:
			// Match found - send battle server URL and ensure it's flushed
			if err := json.NewEncoder(w).Encode(MatchResponse{
				Status:  "matched",
				Message: "Opponent found!",
				WsUrl:   battleServerURL,
			}); err != nil {
				log.Printf("Error sending match response: %v", err)
				return
			}
			w.(http.Flusher).Flush()

		case <-time.After(matchTimeout):
			// Timeout - clean up and notify player
			matchMutex.Lock()
			waitingPlayer = nil
			matchMutex.Unlock()
			if err := json.NewEncoder(w).Encode(MatchResponse{
				Status:  "timeout",
				Message: "No opponent found. Please try again.",
			}); err != nil {
				log.Printf("Error sending timeout response: %v", err)
				return
			}
			w.(http.Flusher).Flush()
		}
	} else {
		// Second player - signal waiting player and send battle server URL
		close(waitingPlayer)
		waitingPlayer = nil
		matchMutex.Unlock()

		if err := json.NewEncoder(w).Encode(MatchResponse{
			Status:  "matched",
			Message: "Opponent found!",
			WsUrl:   battleServerURL,
		}); err != nil {
			log.Printf("Error sending match response: %v", err)
			return
		}
		w.(http.Flusher).Flush()
	}
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
