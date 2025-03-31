# 猜数字游戏

这是一个简单的在线双人猜数字游戏。游戏会随机生成一个1-100之间的数字，两个玩家轮流猜测，谁猜中了谁就输。每次猜测后，系统会更新当前可猜测的范围。

## 项目结构

```
guess-number-game/
├── web/                  # 前端静态文件
│   ├── index.html       # 主页面
│   ├── css/            
│   │   └── style.css   # 样式文件
│   └── js/
│       └── main.js     # 前端逻辑
├── lobby-server/         # 大厅服务器
│   ├── main.go         # 玩家匹配服务
│   └── go.mod          # Go模块文件
└── battle-server/        # 战斗服务器
    ├── main.go         # 游戏逻辑服务
    └── go.mod          # Go模块文件
```

## 技术栈

- 前端：HTML + CSS + JavaScript
- 后端：Go
- 通信：HTTP + WebSocket

## 运行要求

- Go 1.16+
- Nginx（或其他Web服务器，用于部署静态文件）

## 安装和运行

1. 克隆项目：
```bash
git clone [repository-url]
cd guess-number-game
```

2. 启动战斗服务器：
```bash
cd battle-server
go run main.go
```
战斗服务器将在 8081 端口启动。

3. 启动大厅服务器：
```bash
cd lobby-server
go run main.go
```
大厅服务器将在 8080 端口启动。

4. 部署静态文件：

使用 Nginx 配置示例：
```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        root /path/to/guess-number-game/web;
        index index.html;
    }
}
```

## 游戏规则

1. 打开游戏页面，点击"开始游戏"按钮进入匹配队列
2. 等待另一个玩家加入游戏
3. 匹配成功后，系统随机选择先手玩家
4. 轮到自己时，在输入框中输入猜测的数字（1-100）
5. 每次猜测后，系统会更新可猜测的范围
6. 谁猜中了目标数字，谁就输掉游戏
7. 如果对手断开连接，剩余玩家自动获胜

## 通信协议

### 大厅服务器 API

- 匹配请求：
  - 端点：`GET /match`
  - 响应：`{"wsUrl": "ws://localhost:8081/game"}`

### 战斗服务器 WebSocket 消息

1. 等待消息（第一个玩家连接时）：
```json
{
    "type": "waiting",
    "message": "等待其他玩家加入..."
}
```

2. 游戏开始消息（第二个玩家加入后）：
```json
{
    "type": "game_start",
    "isFirstPlayer": true,
    "minNumber": 1,
    "maxNumber": 100
}
```

2. 猜测请求：
```json
{
    "type": "guess",
    "number": 50
}
```

3. 猜测结果：
```json
{
    "type": "guess_result",
    "minNumber": 1,
    "maxNumber": 49,
    "nextTurn": true,
    "gameOver": false
}
```

4. 游戏结束：
```json
{
    "type": "guess_result",
    "gameOver": true,
    "winner": true,
    "number": 42,
    "message": "正确数字是：42"
}
```

## 开发说明

- 前端默认连接到 `localhost:8080` 作为大厅服务器
- 战斗服务器地址在大厅服务器中配置为常量
- 所有服务器都配置了 CORS，支持跨域请求
- WebSocket 连接支持自动重连机制

## 注意事项

- 确保防火墙允许 8080 和 8081 端口的访问
- 建议在生产环境中配置 HTTPS 和 WSS
- 可以根据需要修改服务器地址和端口配置
