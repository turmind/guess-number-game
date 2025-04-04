package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/gamelift"
	"github.com/aws/aws-sdk-go-v2/service/gamelift/types"
)

const (
	defaultPort  = 8080
	matchTimeout = 180 * time.Second
)

var (
	waitingPlayer  chan *types.GameSession // Channel to signal waiting player and share game session
	matchMutex     sync.Mutex
	gameLiftClient *gamelift.Client
	fleetID        string
	awsRegion      string
	location       string
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

func createGameSession(ctx context.Context) (*types.GameSession, error) {
	input := &gamelift.CreateGameSessionInput{
		FleetId:                   &fleetID,
		MaximumPlayerSessionCount: aws.Int32(2),
		Location:                  &location,
		GameProperties: []types.GameProperty{
			{
				Key:   aws.String("exampleProperty"),
				Value: aws.String("exampleValue"),
			},
		},
	}

	log.Printf("Creating game session with input: %+v", input)
	result, err := gameLiftClient.CreateGameSession(ctx, input)
	if err != nil {
		log.Printf("Error creating game session: %v", err)
		return nil, fmt.Errorf("failed to create game session: %v", err)
	}

	log.Printf("Game session created successfully: %+v", result.GameSession)
	return result.GameSession, nil
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
		waitingPlayer = make(chan *types.GameSession)
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
		case gameSession, ok := <-waitingPlayer:
			if !ok {
				// Channel was closed due to error
				if err := json.NewEncoder(w).Encode(MatchResponse{
					Status:  "error",
					Message: "Failed to create game session",
				}); err != nil {
					log.Printf("Error sending error response to first player: %v", err)
				}
				return
			}
			// Send battle server URL
			wsUrl := fmt.Sprintf("ws://%s:%d/game", *gameSession.IpAddress, *gameSession.Port)
			if err := json.NewEncoder(w).Encode(MatchResponse{
				Status:  "matched",
				Message: "Opponent found!",
				WsUrl:   wsUrl,
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
		// Second player - create game session and signal waiting player
		gameSession, err := createGameSession(r.Context())
		if err != nil {
			log.Printf("Error creating game session: %v", err)
			// Close channel to notify waiting player about error
			close(waitingPlayer)
			waitingPlayer = nil
			matchMutex.Unlock()

			// Send error response to second player
			if err := json.NewEncoder(w).Encode(MatchResponse{
				Status:  "error",
				Message: "Failed to create game session",
			}); err != nil {
				log.Printf("Error sending error response to second player: %v", err)
			}
			return
		}

		// Send game session to waiting player and close channel
		waitingPlayer <- gameSession
		close(waitingPlayer)
		waitingPlayer = nil
		matchMutex.Unlock()

		// Send battle server URL to second player
		wsUrl := fmt.Sprintf("ws://%s:%d/game", *gameSession.IpAddress, *gameSession.Port)
		if err := json.NewEncoder(w).Encode(MatchResponse{
			Status:  "matched",
			Message: "Opponent found!",
			WsUrl:   wsUrl,
		}); err != nil {
			log.Printf("Error sending match response: %v", err)
			return
		}
		w.(http.Flusher).Flush()
	}
}

func main() {
	port := flag.Int("port", defaultPort, "Port to run the lobby server on")
	fleetIDFlag := flag.String("fleet-id", "", "AWS GameLift Fleet ID")
	awsRegionFlag := flag.String("region", "ap-southeast-1", "AWS Region")
	locationFlag := flag.String("location", "custom-location-1", "AWS GameLift Location")
	flag.Parse()

	if *fleetIDFlag == "" {
		log.Fatal("Fleet ID is required")
	}

	fleetID = *fleetIDFlag
	awsRegion = *awsRegionFlag
	location = *locationFlag

	// Initialize AWS config
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(awsRegion),
	)
	if err != nil {
		log.Fatalf("Unable to load AWS config: %v", err)
	}

	// Create GameLift client
	gameLiftClient = gamelift.NewFromConfig(cfg)

	addr := fmt.Sprintf(":%d", *port)
	http.HandleFunc("/match", match)

	log.Printf("Lobby server starting on %s (Fleet ID: %s, Region: %s, Location: %s)",
		addr, fleetID, awsRegion, location)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
