package main

import (
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
		http.Error(w, "游戏已满", http.StatusServiceUnavailable)
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
		conn.WriteJSON(Message{Type: "waiting", Message: "等待其他玩家加入..."})
		return
	}

	// 创建新游戏
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

	log.Printf("新游戏开始！目标数字: %d, 玩家 %d 先手", game.targetNum, game.currentIdx)

	// 通知游戏开始
	for i, p := range game.players {
		p.conn.WriteJSON(Message{
			Type:    "start",
			Message: fmt.Sprintf("游戏开始！%s", map[bool]string{true: "你是先手", false: "对手先手"}[i == game.currentIdx]),
		})
	}

	// 开始游戏
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
		g.players[playerIdx].conn.WriteJSON(Message{Type: "error", Message: "不是你的回合"})
		return
	}

	if guess < g.minNumber || guess > g.maxNumber {
		g.players[playerIdx].conn.WriteJSON(Message{Type: "error", Message: "猜测的数字超出范围"})
		return
	}

	if guess == g.targetNum {
		log.Printf("游戏结束！目标数字: %d, 玩家 %d 猜中并输掉了游戏", g.targetNum, playerIdx)
		for i, p := range g.players {
			p.conn.WriteJSON(Message{
				Type:    "end",
				Message: fmt.Sprintf("游戏结束！正确数字是：%d。%s", g.targetNum, map[bool]string{true: "你赢了！", false: "你输了！"}[i != playerIdx]),
			})
		}
		g.close()
		os.Exit(0)
	}

	// 更新范围并切换玩家
	if guess < g.targetNum {
		g.minNumber = guess + 1
	} else {
		g.maxNumber = guess - 1
	}
	g.currentIdx = 1 - g.currentIdx

	// 通知结果
	for i, p := range g.players {
		p.conn.WriteJSON(Message{
			Type:    "update",
			Message: fmt.Sprintf("可猜测范围：%d-%d。%s", g.minNumber, g.maxNumber, map[bool]string{true: "轮到你猜测", false: "等待对手猜测"}[i == g.currentIdx]),
		})
	}
}

func (g *Game) handleDisconnect(playerIdx int) {
	g.mu.Lock()
	defer g.mu.Unlock()

	log.Printf("玩家 %d 断开连接", playerIdx)
	otherIdx := 1 - playerIdx
	if g.players[otherIdx] != nil {
		g.players[otherIdx].conn.WriteJSON(Message{
			Type:    "end",
			Message: "对手已断开连接，你赢了！",
		})
	}
	g.close()
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
	http.HandleFunc("/game", handleGame)
	log.Println("Battle server starting on :8081")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal(err)
	}
}
