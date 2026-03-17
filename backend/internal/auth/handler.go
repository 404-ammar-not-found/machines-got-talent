package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler wires HTTP routes to the auth service.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts auth endpoints onto the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/register", h.Register)
	rg.POST("/login", h.Login)
	rg.POST("/reset-password", h.ResetPassword)
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.svc.Register(req)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmailTaken), errors.Is(err, ErrUsernameTaken):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "registration failed"})
		}
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.svc.Login(req)
	if err != nil {
		switch {
		case errors.Is(err, ErrUserNotFound), errors.Is(err, ErrInvalidPassword):
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		}
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.svc.ResetPassword(req)
	if err != nil {
		// Don't reveal whether the email exists.
		c.JSON(http.StatusOK, gin.H{"message": "if that email is registered, a reset token has been issued"})
		return
	}
	c.JSON(http.StatusOK, resp)
}
