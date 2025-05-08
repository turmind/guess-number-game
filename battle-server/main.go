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
	players      [2]*Player
	targetNum    int
	minNumber    int
	maxNumber    int
	currentIdx   int
	mu           sync.Mutex
	isEven       int
	sum          int
	isPrime      int
	turnTimer    *time.Timer
	turnDeadline time.Time
}

type Message struct {
	Type    string `json:"type"`
	Number  int    `json:"number,omitempty"`
	Message string `json:"message,omitempty"`
	IsEven  int    `json:"isEven"`
	Sum     int    `json:"sum"`
	IsPrime int    `json:"isPrime"`
}

func isPrime(n int) bool {
	if n <= 1 {
		return false
	}
	for i := 2; i*i <= n; i++ {
		if n%i == 0 {
			return false
		}
	}
	return true
}

func sumDigits(n int) int {
	sum := 0
	for n > 0 {
		sum += n % 10
		n /= 10
	}
	return sum
}

func getHints(num int) (int, int, int) {
	// Calculate all properties
	isEvenVal := 0
	if num%2 == 0 {
		isEvenVal = 1
	}

	sumVal := sumDigits(num)

	isPrimeVal := 0
	if isPrime(num) {
		isPrimeVal = 1
	}

	// Randomly select 1-2 hints to show
	hints := []int{0, 1, 2} // 0: isEven, 1: sum, 2: isPrime
	rand.Shuffle(len(hints), func(i, j int) { hints[i], hints[j] = hints[j], hints[i] })

	numHints := rand.Intn(2) + 1 // Random number between 1 and 2
	selectedHints := make(map[int]bool)
	for i := 0; i < numHints; i++ {
		selectedHints[hints[i]] = true
	}

	// Set values based on selection
	isEven := -1
	sum := -1
	isPrime := -1

	if selectedHints[0] {
		isEven = isEvenVal
	}
	if selectedHints[1] {
		sum = sumVal
	}
	if selectedHints[2] {
		isPrime = isPrimeVal
	}

	return isEven, sum, isPrime
}

var (
	waitingPlayer     *Player
	waitingPlayerTime time.Time
	serverFull        bool
	mu                sync.Mutex
	waitingTimer      *time.Timer
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func handleWaitingTimeout() {
	mu.Lock()
	defer mu.Unlock()

	// Check if there's still a waiting player
	if waitingPlayer != nil {
		log.Println("Timeout: No opponent connected within 10 seconds")
		waitingPlayer.conn.WriteJSON(Message{
			Type:    "end",
			Message: "No opponent connected within 10 seconds. You win by default!",
		})
		waitingPlayer.conn.Close()
		waitingPlayer = nil

		// Exit the server after timeout
		log.Println("Exiting server due to waiting timeout")
		os.Exit(0)
	}
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
		waitingPlayerTime = time.Now()

		// Set a timeout for the waiting player
		if waitingTimer != nil {
			waitingTimer.Stop()
		}
		waitingTimer = time.AfterFunc(10*time.Second, handleWaitingTimeout)

		mu.Unlock()
		conn.WriteJSON(Message{Type: "waiting", Message: "Waiting for another player..."})
		return
	}

	// Cancel the timeout since a second player has connected
	if waitingTimer != nil {
		waitingTimer.Stop()
	}

	targetNum := rand.Intn(100) + 1
	isEven, sum, isPrime := getHints(targetNum)

	// Create new game
	game := &Game{
		players:    [2]*Player{waitingPlayer, player},
		targetNum:  targetNum,
		minNumber:  1,
		maxNumber:  100,
		currentIdx: rand.Intn(2),
		isEven:     isEven,
		sum:        sum,
		isPrime:    isPrime,
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
			IsEven:  isEven,
			Sum:     sum,
			IsPrime: isPrime,
		})
	}

	// Start game
	game.run()
}

func (g *Game) startTurnTimer() {
	// Cancel any existing timer
	if g.turnTimer != nil {
		g.turnTimer.Stop()
		g.turnTimer = nil
	}

	// Set a 30-second deadline for the current player's turn
	g.turnDeadline = time.Now().Add(30 * time.Second)
	g.turnTimer = time.AfterFunc(30*time.Second, func() {
		g.handleTurnTimeout()
	})
}

func (g *Game) handleTurnTimeout() {
	g.mu.Lock()
	defer g.mu.Unlock()

	log.Printf("Turn timeout: Player %d took too long to make a guess", g.currentIdx)

	// The current player loses, the other player wins
	winnerIdx := 1 - g.currentIdx

	for i, p := range g.players {
		if p != nil && p.conn != nil {
			p.conn.WriteJSON(Message{
				Type: "end",
				Message: fmt.Sprintf("Game over! Player %d took too long to make a guess. %s",
					g.currentIdx,
					map[bool]string{true: "You win!", false: "You lose!"}[i == winnerIdx]),
			})
		}
	}

	g.close()
	log.Println("Game finished due to turn timeout, exiting server")
	mu.Lock()
	serverFull = false
	mu.Unlock()
	os.Exit(0)
}

func (g *Game) run() {
	// Start the turn timer for the first player
	g.startTurnTimer()

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

	if playerIdx != g.currentIdx {
		g.players[playerIdx].conn.WriteJSON(Message{Type: "error", Message: "Not your turn"})
		g.mu.Unlock()
		return
	}

	// Cancel the turn timer since the player has made a guess
	if g.turnTimer != nil {
		g.turnTimer.Stop()
		g.turnTimer = nil
	}

	if guess < g.minNumber || guess > g.maxNumber {
		g.players[playerIdx].conn.WriteJSON(Message{Type: "error", Message: "Number out of valid range"})
		g.mu.Unlock()
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

	// Send result with stored hints
	for i, p := range g.players {
		p.conn.WriteJSON(Message{
			Type:    "update",
			Message: fmt.Sprintf("Valid range: %d-%d. %s", g.minNumber, g.maxNumber, map[bool]string{true: "It's your turn", false: "Opponent's turn"}[i == g.currentIdx]),
			IsEven:  g.isEven,
			Sum:     g.sum,
			IsPrime: g.isPrime,
		})
	}

	// Start the timer for the next player's turn after sending messages
	g.startTurnTimer()

	g.mu.Unlock()
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
