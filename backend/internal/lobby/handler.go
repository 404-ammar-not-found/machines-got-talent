package lobby

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/machines-got-talent/backend/internal/auth"
)

// Handler wires HTTP routes to the lobby service.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts lobby endpoints (all require auth middleware applied upstream).
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/create", h.Create)
	rg.GET("/list", h.List)
	rg.POST("/join", h.Join)
	rg.POST("/leave", h.Leave)
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateLobbyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lobby, err := h.svc.CreateLobby(auth.GetUserID(c), auth.GetUsername(c), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create lobby"})
		return
	}

	lobby.RLock()
	resp := toLobbyResponse(lobby)
	lobby.RUnlock()
	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) List(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"lobbies": h.svc.ListLobbies()})
}

func (h *Handler) Join(c *gin.Context) {
	var req JoinLobbyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lobby, err := h.svc.JoinLobby(auth.GetUserID(c), auth.GetUsername(c), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrLobbyNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case errors.Is(err, ErrWrongPassword):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		case errors.Is(err, ErrAlreadyJoined):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case errors.Is(err, ErrGameStarted):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not join lobby"})
		}
		return
	}

	lobby.RLock()
	resp := toLobbyResponse(lobby)
	lobby.RUnlock()
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Leave(c *gin.Context) {
	var req LeaveLobbyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	closed, err := h.svc.LeaveLobby(auth.GetUserID(c), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrLobbyNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case errors.Is(err, ErrNotInLobby):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not leave lobby"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"lobby_closed": closed})
}
