package main

import (
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/machines-got-talent/backend/internal/ai"
	"github.com/machines-got-talent/backend/internal/auth"
	"github.com/machines-got-talent/backend/internal/db"
	"github.com/machines-got-talent/backend/internal/game"
	"github.com/machines-got-talent/backend/internal/lobby"
	"github.com/machines-got-talent/backend/pkg/config"
)

func main() {
	// Initialize database
	db.InitDB()

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

	// CORS configuration
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

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
