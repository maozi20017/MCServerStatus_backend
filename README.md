# Minecraft 伺服器狀態查詢

這是一個用 Go 語言編寫的 Web 服務，用於查詢 Minecraft 伺服器的狀態。它使用 Gin 框架來處理 HTTP 請求，並實現了 Minecraft 伺服器查詢協議來獲取伺服器資訊。

## 功能特點

- 查詢 Minecraft 伺服器狀態
- 支援自定義端口
- 處理各種伺服器回應格式
- 使用 Gin 框架提供 RESTful API

## 安裝

確保您已安裝 Go（建議版本 1.16 或更高）。然後，按照以下步驟進行安裝：

1. 克隆此儲存庫：
   ```
   git clone https://github.com/maozi20017/MCServerStatus_backend
   cd [專案目錄]
   ```

2. 安裝依賴：
   ```
   go mod tidy
   ```

## 使用方法

1. 設定環境變數（可選）：
   - `PORT`: 伺服器監聽的端口（預設為 8080）
   - `GIN_MODE`: Gin 的運行模式（預設為 release）

2. 運行伺服器：
   ```
   go run main.go
   ```

3. 使用 API：
   發送 GET 請求到 `/api/server-status`，並提供 `address` 查詢參數：
   ```
   http://localhost:8080/api/server-status?address=example.minecraft.com
   ```

## API 說明

### GET /api/server-status

查詢 Minecraft 伺服器狀態。

查詢參數：
- `address`: Minecraft 伺服器的地址（必填）

回應範例：
```json
{
  "version": {
    "name": "1.19.2",
    "protocol": 760
  },
  "players": {
    "max": 100,
    "online": 5,
    "sample": [
      {
        "name": "Player1",
        "id": "uuid-here"
      }
    ]
  },
  "description": {
    "text": "Welcome to our Minecraft server!"
  },
  "favicon": "data:image/png;base64,..."
}
```

## 開發

- `main.go`: 應用程式的入口點
- `internal/api/routes.go`: 定義 API 路由
- `internal/api/handlers/server.go`: 處理 API 請求
- `internal/service/server.go`: 實現 Minecraft 伺服器狀態查詢邏輯
