package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
)

var battleServerURL string

type MatchResponse struct {
	WsUrl string `json:"wsUrl"`
}

func handleMatch(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// 处理预检请求
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 只允许GET请求
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 返回战斗服务器地址
	w.Header().Set("Content-Type", "application/json")
	response := MatchResponse{
		WsUrl: battleServerURL,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func main() {
	port := flag.Int("port", 8080, "Port to run the lobby server on")
	battlePort := flag.Int("battle-port", 8081, "Port of the battle server")
	flag.Parse()

	battleServerURL = fmt.Sprintf("ws://localhost:%d/game", *battlePort)

	http.HandleFunc("/match", handleMatch)
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Lobby server starting on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
