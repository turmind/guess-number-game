package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Player struct {
	conn *websocket.Conn
}

type Game struct {
	players    [2]*Player
	targetNum  int
	minNumber  int
	maxNumber  int
	currentIdx int
	mu         sync.Mutex
}

type Message struct {
	Type    string `json:"type"`
	Number  int    `json:"number,omitempty"`
	Message string `json:"message,omitempty"`
}

var (
	waitingPlayer *Player
	serverFull    bool
	mu            sync.Mutex
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func handleGame(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	if serverFull {
		mu.Unlock()
		http.Error(w, "Game is full", http.StatusServiceUnavailable)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		mu.Unlock()
		return
	}

	player := &Player{conn: conn}

	if waitingPlayer == nil {
		waitingPlayer = player
		mu.Unlock()
		conn.WriteJSON(Message{Type: "waiting", Message: "Waiting for another player..."})
		return
	}

	// Create new game
	game := &Game{
		players:    [2]*Player{waitingPlayer, player},
		targetNum:  rand.Intn(100) + 1,
		minNumber:  1,
		maxNumber:  100,
		currentIdx: rand.Intn(2),
	}
	waitingPlayer = nil
	serverFull = true
	mu.Unlock()

	log.Printf("Game started! Target number: %d, Player %d goes first", game.targetNum, game.currentIdx)

	// Notify game start
	for i, p := range game.players {
		p.conn.WriteJSON(Message{
			Type:    "start",
			Message: fmt.Sprintf("Game started! %s", map[bool]string{true: "It's your turn", false: "Opponent's turn"}[i == game.currentIdx]),
		})
	}

	// Start game
	game.run()
}

func (g *Game) run() {
	for i, player := range g.players {
		go func(idx int, p *Player) {
			for {
				var msg Message
				if err := p.conn.ReadJSON(&msg); err != nil {
					g.handleDisconnect(idx)
					return
				}

				if msg.Type == "guess" {
					g.handleGuess(idx, msg.Number)
				}
			}
		}(i, player)
	}
}

func (g *Game) handleGuess(playerIdx, guess int) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if playerIdx != g.currentIdx {
		g.players[playerIdx].conn.WriteJSON(Message{Type: "error", Message: "Not your turn"})
		return
	}

	if guess < g.minNumber || guess > g.maxNumber {
		g.players[playerIdx].conn.WriteJSON(Message{Type: "error", Message: "Number out of valid range"})
		return
	}

	if guess == g.targetNum {
		log.Printf("Game over! Target number: %d, Player %d guessed it and won!", g.targetNum, playerIdx)
		for i, p := range g.players {
			p.conn.WriteJSON(Message{
				Type:    "end",
				Message: fmt.Sprintf("Game over! Number was: %d. %s", g.targetNum, map[bool]string{true: "You win!", false: "You lose!"}[i == playerIdx]),
			})
		}
		g.close()
		log.Println("Game finished, exiting server")
		mu.Lock()
		serverFull = false
		mu.Unlock()
		os.Exit(0)
	}

	// Update range and switch player
	if guess < g.targetNum {
		g.minNumber = guess + 1
	} else {
		g.maxNumber = guess - 1
	}
	g.currentIdx = 1 - g.currentIdx

	// Send result
	for i, p := range g.players {
		p.conn.WriteJSON(Message{
			Type:    "update",
			Message: fmt.Sprintf("Valid range: %d-%d. %s", g.minNumber, g.maxNumber, map[bool]string{true: "It's your turn", false: "Opponent's turn"}[i == g.currentIdx]),
		})
	}
}

func (g *Game) handleDisconnect(playerIdx int) {
	g.mu.Lock()
	defer g.mu.Unlock()

	log.Printf("Player %d disconnected", playerIdx)
	otherIdx := 1 - playerIdx
	if g.players[otherIdx] != nil {
		g.players[otherIdx].conn.WriteJSON(Message{
			Type:    "end",
			Message: "Opponent disconnected. You win!",
		})
	}
	g.close()
	log.Println("Player disconnected, exiting server")
	mu.Lock()
	serverFull = false
	mu.Unlock()
	os.Exit(0)
}

func (g *Game) close() {
	for _, p := range g.players {
		if p != nil && p.conn != nil {
			p.conn.Close()
		}
	}
}

func main() {
	port := flag.Int("port", 8081, "Port to run the battle server on")
	flag.Parse()

	http.HandleFunc("/game", handleGame)
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Battle server starting on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
