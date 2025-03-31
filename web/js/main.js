// 常量定义
const LOBBY_SERVER = 'http://localhost:8080';

// DOM 元素
const startGameBtn = document.getElementById('startGame');
const gameStatus = document.getElementById('gameStatus');
const gameArea = document.getElementById('gameArea');
const guessInput = document.getElementById('guessInput');
const submitGuessBtn = document.getElementById('submitGuess');
const rangeDisplay = document.getElementById('range');

// 游戏状态
let gameState = {
    ws: null,
    canGuess: false
};

// 更新状态显示
function updateStatus(status) {
    gameStatus.textContent = status;
}

// 启用/禁用猜测输入
function setGuessInputEnabled(enabled) {
    guessInput.disabled = !enabled;
    submitGuessBtn.disabled = !enabled;
    if (enabled) {
        guessInput.focus();
    }
}

// 重置游戏界面
function resetGame() {
    gameArea.classList.add('hidden');
    rangeDisplay.textContent = '可猜测范围：1-100';
    guessInput.value = '';
    setGuessInputEnabled(false);
}

// 开始游戏
startGameBtn.addEventListener('click', async () => {
    startGameBtn.disabled = true;
    updateStatus('正在匹配玩家...');
    resetGame();

    try {
        const response = await fetch(`${LOBBY_SERVER}/match`);
        if (!response.ok) {
            throw new Error('匹配服务器错误');
        }
        const data = await response.json();
        
        if (data.wsUrl) {
            connectToBattleServer(data.wsUrl);
        } else {
            throw new Error('无效的服务器地址');
        }
    } catch (error) {
        console.error('Error:', error);
        updateStatus('匹配失败，请重试');
        startGameBtn.disabled = false;
    }
});

// 连接到战斗服务器
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
            const message = JSON.parse(event.data);
            handleGameMessage(message);
        } catch (error) {
            console.error('Message parsing error:', error);
            updateStatus('消息处理错误');
        }
    };

    gameState.ws.onclose = () => {
        if (!gameState.gameOver) {
            updateStatus('连接已断开，请重试');
            startGameBtn.disabled = false;
        }
    };

    gameState.ws.onerror = () => {
        updateStatus('连接错误，请重试');
        startGameBtn.disabled = false;
    };
}

// 处理游戏消息
function handleGameMessage(message) {
    switch (message.type) {
        case 'waiting':
            gameArea.classList.add('hidden');
            updateStatus(message.message);
            break;
            
        case 'start':
            gameArea.classList.remove('hidden');
            updateStatus(message.message);
            rangeDisplay.textContent = '可猜测范围：1-100';  // 初始化范围显示
            gameState.canGuess = message.message.includes('你是先手');
            setGuessInputEnabled(gameState.canGuess);
            break;

        case 'update':
            updateStatus(message.message);
            // 从消息中提取范围
            const rangeMatch = message.message.match(/范围：(\d+)-(\d+)/);
            if (rangeMatch) {
                rangeDisplay.textContent = `可猜测范围：${rangeMatch[1]}-${rangeMatch[2]}`;
            }
            gameState.canGuess = message.message.includes('轮到你猜测');
            setGuessInputEnabled(gameState.canGuess);
            break;

        case 'end':
            gameState.gameOver = true;
            updateStatus(message.message);
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

// 提交猜测
submitGuessBtn.addEventListener('click', () => {
    const guess = parseInt(guessInput.value);
    if (isNaN(guess) || guess < 1 || guess > 100) {
        alert('请输入1-100之间的数字');
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
