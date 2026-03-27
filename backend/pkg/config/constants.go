package config

import "time"

const (
	// JWT
	JWTSecret = "change-me-in-production-use-env-var"
	JWTExpiry = 24 * time.Hour

	// Server
	ServerPort = ":8080"

	// Python AI service
	AIServiceBaseURL = "http://localhost:8000"

	// Token economy
	FirstPickCost        = 50
	WinnerReward         = 100
	SurvivorRewardMax    = 50
	ParticipationReward  = 5

	// Game constants
	AIStartingScore    = 2
	BuzzerPoints       = -1
	GoldenBuzzerPoints = 2
)
