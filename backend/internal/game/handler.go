package game

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/machines-got-talent/backend/internal/ai"
	"github.com/machines-got-talent/backend/internal/auth"
	"github.com/machines-got-talent/backend/internal/lobby"
	"github.com/machines-got-talent/backend/pkg/config"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Handler struct {
	gameStore  *Store
	lobbySvc   *lobby.Service
	aiClient   *ai.Client
}

func NewHandler(gs *Store, ls *lobby.Service, ac *ai.Client) *Handler {
	return &Handler{gameStore: gs, lobbySvc: ls, aiClient: ac}
}

func (h *Handler) RegisterRoutes(protected *gin.RouterGroup, public *gin.RouterGroup) {
	protected.POST("/start", h.StartGame)
	protected.POST("/use-advantage", h.UseAdvantage)
	public.GET("/ws/:lobbyCode", h.ServeWS)
}

func (h *Handler) StartGame(c *gin.Context) {
	var req struct {
		LobbyCode string `json:"lobby_code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	callerID := auth.GetUserID(c)
	l, err := h.lobbySvc.GetLobby(req.LobbyCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	l.RLock()
	isHost := l.HostID == callerID
	alreadyStarted := l.State != lobby.GameStateWaiting
	l.RUnlock()

	if !isHost {
		c.JSON(http.StatusForbidden, gin.H{"error": "only the host can start the game"})
		return
	}
	if alreadyStarted {
		c.JSON(http.StatusConflict, gin.H{"error": "game has already started"})
		return
	}

	if err := h.lobbySvc.SetGameState(callerID, req.LobbyCode, lobby.GameStateDraft); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if _, err := h.gameStore.StartGame(l, h.aiClient); err != nil {
		h.lobbySvc.SetGameState(callerID, req.LobbyCode, lobby.GameStateWaiting)
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to create AI comedians: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "game started", "lobby_code": req.LobbyCode})
}

func (h *Handler) UseAdvantage(c *gin.Context) {
	var req struct {
		LobbyCode    string `json:"lobby_code"    binding:"required"`
		Advantage    string `json:"advantage"     binding:"required"`
		TokenBalance int    `json:"token_balance" binding:"required,min=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	playerID := auth.GetUserID(c)
	l, err := h.lobbySvc.GetLobby(req.LobbyCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	l.Lock()
	p, exists := l.Players[playerID]
	if exists {
		p.TokenBalance = req.TokenBalance
	}
	l.Unlock()

	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "you are not in this lobby"})
		return
	}

	switch req.Advantage {
	case "first_pick":
		delta, err := h.lobbySvc.ApplyFirstPickAdvantage(playerID, req.LobbyCode, config.FirstPickCost)
		if err != nil {
			c.JSON(http.StatusPaymentRequired, gin.H{"error": err.Error()})
			return
		}
		l.RLock()
		newBalance := l.Players[playerID].TokenBalance
		l.RUnlock()
		c.JSON(http.StatusOK, gin.H{
			"advantage":   "first_pick",
			"delta":       delta,
			"new_balance": newBalance,
		})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown advantage type"})
	}
}

func (h *Handler) ServeWS(c *gin.Context) {
	lobbyCode := c.Param("lobbyCode")
	tokenStr := c.Query("token")
	if tokenStr == "" {
		tokenStr = strings.TrimPrefix(c.GetHeader("Sec-WebSocket-Protocol"), "Bearer ")
	}
	if tokenStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}

	claims, err := auth.ValidateJWT(tokenStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	l, err := h.lobbySvc.GetLobby(lobbyCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "lobby not found"})
		return
	}
	l.RLock()
	_, inLobby := l.Players[claims.UserID]
	l.RUnlock()
	if !inLobby {
		c.JSON(http.StatusForbidden, gin.H{"error": "you are not in this lobby"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	// GET THE HUB FROM THE MANAGER
	m := h.gameStore.GetOrCreate(lobbyCode)

	client := &Client{
		hub:       m.hub,
		conn:      conn,
		send:      make(chan WSMessage, 64),
		PlayerID:  claims.UserID,
		Username:  claims.Username,
		LobbyCode: lobbyCode,
	}

	if err := h.gameStore.ConnectClient(lobbyCode, client); err != nil {
		conn.WriteJSON(NewWSMessage(EventError, map[string]string{"message": err.Error()}))
		conn.Close()
		return
	}
}