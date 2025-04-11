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
├── client-app/          # Desktop application (Electron)
│   ├── main.js         # Electron main process
│   ├── src/            # Application source
│   │   ├── index.html  # App main page
│   │   ├── css/        # App styles
│   │   └── js/         # App logic
│   └── package.json    # Dependencies and scripts
├── lobby-server/         # Local lobby server (port 8080)
│   ├── main.go         # Returns battle server address
│   └── go.mod          # Go module file
├── lobby-server-gamelift/ # AWS GameLift lobby server (port 8080)
│   ├── main.go         # GameLift matchmaking and session management
│   ├── go.mod          # Go module file
│   └── go.sum          # Go module checksums
└── battle-server/        # Battle server (port 8081)
    ├── main.go         # Game logic
    └── go.mod          # Go module file
```

## Running Instructions

### Using Local Lobby Server

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

3. Choose your preferred client:

   a. Web Client:
   - Deploy the client directory using nginx or simply open index.html in a browser

   b. Desktop Client:
   ```bash
   cd client-app
   npm install
   npm start
   ```

### Using AWS GameLift Lobby Server

1. Start the GameLift lobby server (runs on port 8080):
```bash
cd lobby-server-gamelift
go run main.go --fleet-id <your-fleet-id> --region <aws-region> --location <gamelift-location>
# OR using alias ID
go run main.go --alias-id <your-alias-id> --region <aws-region> --location <gamelift-location>
```

Required flags:
- Either `--fleet-id` or `--alias-id` (but not both)
- `--region`: AWS region (default: ap-southeast-1)
- `--location`: GameLift location (default: custom-location-1)
- Optional: `--port` to specify custom port (default: 8080)

2. Start your preferred client as mentioned above

## Communication Protocol

### Local Lobby Server API

- Match request: `GET /match`
- Response: `{"wsUrl": "ws://localhost:8081/game"}`

### GameLift Lobby Server API

- Match request: `GET /match`
- Response types:
```json
{
    "status": "waiting|matched|timeout|error",
    "message": "Status message",
    "wsUrl": "ws://gameserver:port/game" // Only included when status is "matched"
}
```

Status types:
- waiting: Waiting for another player to join
- matched: Match found, includes WebSocket URL for game server
- timeout: No opponent found within timeout period (180 seconds)
- error: Error occurred during matchmaking

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
6. Strategy is key: The player who guesses the target number WIN!
7. If opponent disconnects, remaining player wins automatically

## Technical Notes

- Native battle server runs on WebSocket port 8081
- Both lobby servers run on HTTP port 8080 (configurable)
- Battle server only allows two players per game
- Server automatically exits after game ends
- Restart battle server to start a new game session
