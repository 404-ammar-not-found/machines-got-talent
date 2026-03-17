package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/machines-got-talent/backend/internal/ai"
	"github.com/machines-got-talent/backend/internal/auth"
	"github.com/machines-got-talent/backend/internal/game"
	"github.com/machines-got-talent/backend/internal/lobby"
	"github.com/machines-got-talent/backend/pkg/config"
)

func main() {
	// --- Dependency wiring ---
	authSvc := auth.NewService()
	authHandler := auth.NewHandler(authSvc)

	lobbyStore := lobby.NewStore()
	lobbySvc := lobby.NewService(lobbyStore)
	lobbyHandler := lobby.NewHandler(lobbySvc)

	gameStore := game.NewStore()
	aiClient := ai.NewClient()
	gameHandler := game.NewHandler(gameStore, lobbySvc, aiClient)

	// --- Router setup ---
	r := gin.Default()

	// Public routes
	authGroup := r.Group("/auth")
	authHandler.RegisterRoutes(authGroup)

	public := r.Group("/")

	// Protected routes (require JWT)
	protected := r.Group("/")
	protected.Use(auth.Middleware())

	lobbyGroup := protected.Group("/lobby")
	lobbyHandler.RegisterRoutes(lobbyGroup)

	gameGroup := protected.Group("/game")
	gameHandler.RegisterRoutes(gameGroup, public)

	// --- Start ---
	log.Printf("AI Comedian backend listening on %s", config.ServerPort)
	if err := r.Run(config.ServerPort); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
