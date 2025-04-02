// DOM Elements
const lobbyServerInput = document.getElementById('lobbyServer');
const saveConfigBtn = document.getElementById('saveConfig');
const startGameBtn = document.getElementById('startGame');

// Load saved lobby server address or use default
const DEFAULT_LOBBY_SERVER = 'http://localhost:8080';
let lobbyServer = localStorage.getItem('lobbyServer') || DEFAULT_LOBBY_SERVER;
lobbyServerInput.value = lobbyServer;

// Save server configuration
saveConfigBtn.addEventListener('click', () => {
    const newAddress = lobbyServerInput.value.trim();
    if (newAddress) {
        lobbyServer = newAddress;
        localStorage.setItem('lobbyServer', lobbyServer);
        updateStatus('Server configuration saved');
    }
});

const gameStatus = document.getElementById('gameStatus');
const gameArea = document.getElementById('gameArea');
const guessInput = document.getElementById('guessInput');
const submitGuessBtn = document.getElementById('submitGuess');
const rangeDisplay = document.getElementById('range');

// Game state
let gameState = {
    ws: null,
    canGuess: false,
    gameOver: false
};

// Update status display
function updateStatus(status) {
    gameStatus.textContent = status;
}

// Enable/disable guess input
function setGuessInputEnabled(enabled) {
    guessInput.disabled = !enabled;
    submitGuessBtn.disabled = !enabled;
    if (enabled) {
        guessInput.focus();
    }
}

// Reset game interface
function resetGame() {
    gameArea.classList.add('hidden');
    rangeDisplay.textContent = 'Valid range: 1-100';
    guessInput.value = '';
    setGuessInputEnabled(false);
    gameState.gameOver = false;
}

// Start game
startGameBtn.addEventListener('click', async () => {
    startGameBtn.disabled = true;
    updateStatus('Finding opponent...');
    resetGame();

    try {
        const response = await fetch(`${lobbyServer}/match`);
        if (!response.ok) {
            throw new Error('Matchmaking server error');
        }

        // Create a reader to handle multiple JSON responses
        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';

        while (true) {
            const {value, done} = await reader.read();
            if (done) break;
            
            buffer += decoder.decode(value, {stream: true});
            
            // Process complete JSON objects
            const lines = buffer.split('\n');
            buffer = lines.pop() || ''; // Keep the incomplete line in buffer
            
            for (const line of lines) {
                if (!line.trim()) continue; // Skip empty lines
                
                try {
                    const data = JSON.parse(line);
                    
                    switch (data.status) {
                        case 'waiting':
                            updateStatus(data.message);
                            break;
                            
                        case 'matched':
                            updateStatus(data.message);
                            if (data.wsUrl) {
                                // Short delay to show the "opponent found" message
                                setTimeout(() => connectToBattleServer(data.wsUrl), 1000);
                                return; // Exit the polling loop
                            } else {
                                throw new Error('Invalid server address');
                            }
                            
                        case 'timeout':
                            updateStatus(data.message);
                            startGameBtn.disabled = false;
                            return; // Exit the polling loop
                            
                        default:
                            throw new Error('Invalid response from server');
                    }
                } catch (parseError) {
                    console.error('Parse error:', parseError);
                    // Continue to next line if this one is invalid
                }
            }
        }
    } catch (error) {
        console.error('Error:', error);
        updateStatus('Matchmaking failed, please try again');
        startGameBtn.disabled = false;
    }
});

// Connect to battle server
function connectToBattleServer(wsUrl) {
    if (gameState.ws) {
        gameState.ws.close();
    }

    gameState.ws = new WebSocket(wsUrl);

    gameState.ws.onopen = () => {
        console.log('Connected to battle server');
    };

    gameState.ws.onmessage = (event) => {
        try {
            // amazonq-ignore-next-line
            const message = JSON.parse(event.data);
            handleGameMessage(message);
        } catch (error) {
            console.error('Message parsing error:', error);
            updateStatus('Message processing error');
        }
    };

    gameState.ws.onclose = () => {
        if (!gameState.gameOver) {
            updateStatus('Connection lost, please try again');
            startGameBtn.disabled = false;
        }
    };

    gameState.ws.onerror = () => {
        updateStatus('Connection error, please try again');
        startGameBtn.disabled = false;
    };
}

// Handle game messages
function handleGameMessage(message) {
    switch (message.type) {
        case 'waiting':
            gameArea.classList.add('hidden');
            updateStatus(message.message);
            break;
            
        case 'start':
            gameArea.classList.remove('hidden');
            updateStatus(message.message);
            rangeDisplay.textContent = 'Valid range: 1-100';
            gameState.canGuess = message.message.includes('your turn');
            setGuessInputEnabled(gameState.canGuess);
            break;

        case 'update':
            updateStatus(message.message);
            // Extract range from message
            const rangeMatch = message.message.match(/range: (\d+)-(\d+)/);
            if (rangeMatch) {
                rangeDisplay.textContent = `Valid range: ${rangeMatch[1]}-${rangeMatch[2]}`;
            }
            gameState.canGuess = message.message.includes('your turn');
            setGuessInputEnabled(gameState.canGuess);
            break;

        case 'end':
            gameState.gameOver = true;
            if (message.message.includes('You win!')) {
                updateStatus('🎉 ' + message.message + ' 🎉');
                gameStatus.style.color = '#2ecc71';
                gameStatus.style.fontSize = '24px';
                gameStatus.style.fontWeight = 'bold';
            } else {
                updateStatus(message.message);
                gameStatus.style.color = '#e74c3c';
            }
            gameArea.classList.add('hidden');
            startGameBtn.disabled = false;
            break;

        case 'error':
            updateStatus(message.message);
            break;

        default:
            console.error('Unknown message type:', message.type);
            break;
    }
}

// Submit guess
submitGuessBtn.addEventListener('click', () => {
    const guess = parseInt(guessInput.value);
    if (isNaN(guess) || guess < 1 || guess > 100) {
        // amazonq-ignore-next-line
        alert('Please enter a number between 1 and 100');
        return;
    }

    if (gameState.ws && gameState.ws.readyState === WebSocket.OPEN) {
        gameState.ws.send(JSON.stringify({
            type: 'guess',
            number: guess
        }));
        guessInput.value = '';
        setGuessInputEnabled(false);
    }
});
