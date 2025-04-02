# Number Guessing Game

A simple online two-player number guessing game. The game randomly generates a number between 1-100, and players take turns guessing. Unique twist: The player who guesses the correct number loses!

## Project Structure

```
guess-number-game/
├── client/               # Frontend static files
│   ├── index.html       # Main page
│   ├── css/            
│   │   └── style.css   # Styles
│   └── js/
│       └── main.js     # Frontend logic
├── lobby-server/         # Lobby server (port 8080)
│   ├── main.go         # Returns battle server address
│   └── go.mod          # Go module file
└── battle-server/        # Battle server (port 8081)
    ├── main.go         # Game logic
    └── go.mod          # Go module file
```

## Running Instructions

1. Start the battle server (runs on port 8081):
```bash
cd battle-server
go run main.go
```

2. Start the lobby server (runs on port 8080):
```bash
cd lobby-server
go run main.go
```

3. Deploy the client directory using nginx or simply open index.html in a browser

## Communication Protocol

### Lobby Server API

- Match request: `GET /match`
- Response: `{"wsUrl": "ws://localhost:8081/game"}`

### Battle Server WebSocket Messages

1. Client sends:
```json
{
    "type": "guess",
    "number": 50
}
```

2. Server sends:
```json
{
    "type": "waiting|start|update|end|error",
    "message": "Status message"
}
```

Message types:
- waiting: Waiting for another player
- start: Game starts, includes first player info
- update: Game progress, includes range update and turn info
- end: Game over, includes win/lose result
- error: Error message

## Game Rules

1. Open the game page, click "Start Game" to enter matchmaking
2. Wait for another player to join
3. Game starts with a randomly chosen first player
4. When it's your turn, enter your guess (1-100)
5. After each guess, the valid range is updated to help players
6. Strategy is key: The player who guesses the target number LOSES!
7. If opponent disconnects, remaining player wins automatically

## Technical Notes

- Battle server runs on WebSocket port 8081
- Lobby server runs on HTTP port 8080
- Battle server only allows two players per game
- Server automatically exits after game ends
- Restart battle server to start a new game session
