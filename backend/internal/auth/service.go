package auth

import (
	"database/sql"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/machines-got-talent/backend/internal/db"
	"github.com/machines-got-talent/backend/pkg/config"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrInvalidPassword = errors.New("invalid password")
	ErrEmailTaken      = errors.New("email already registered")
	ErrUsernameTaken   = errors.New("username already taken")
)

// Service provides auth business logic via MySQL.
type Service struct {
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Register(req RegisterRequest) (*AuthResponse, error) {
	// 1. Check if email exists
	var existingID string
	err := db.DB.QueryRow("SELECT id FROM users WHERE email = ?", req.Email).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if err == nil {
		return nil, ErrEmailTaken
	}

	// 2. Check if username exists
	err = db.DB.QueryRow("SELECT id FROM users WHERE username = ?", req.Username).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if err == nil {
		return nil, ErrUsernameTaken
	}

	// 3. Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 4. Create user
	user := &User{
		ID:           uuid.NewString(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hash),
		WinCount:     0,
		Balance:      0,
	}

	_, err = db.DB.Exec(
		"INSERT INTO users (id, username, email, password_hash, win_count, balance) VALUES (?, ?, ?, ?, ?, ?)",
		user.ID, user.Username, user.Email, user.PasswordHash, user.WinCount, user.Balance,
	)
	if err != nil {
		return nil, err
	}

	token, err := generateJWT(user)
	if err != nil {
		return nil, err
	}
	return &AuthResponse{Token: token, User: *user}, nil
}

func (s *Service) Login(req LoginRequest) (*AuthResponse, error) {
	var user User
	err := db.DB.QueryRow(
		"SELECT id, username, email, password_hash, win_count, balance FROM users WHERE email = ?",
		req.Email,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.WinCount, &user.Balance)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidPassword
	}

	token, err := generateJWT(&user)
	if err != nil {
		return nil, err
	}
	return &AuthResponse{Token: token, User: user}, nil
}

func (s *Service) GetUserByID(id string) (*User, error) {
	var user User
	err := db.DB.QueryRow(
		"SELECT id, username, email, password_hash, win_count, balance FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.WinCount, &user.Balance)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// ResetPassword generates a reset token (email delivery not implemented).
func (s *Service) ResetPassword(req ResetPasswordRequest) (*ResetPasswordResponse, error) {
	// 1. Check if user exists
	var id string
	err := db.DB.QueryRow("SELECT id FROM users WHERE email = ?", req.Email).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	resetToken := uuid.NewString()
	return &ResetPasswordResponse{
		ResetToken: resetToken,
		Message:    "Use this token to reset your password (email delivery not yet implemented).",
	}, nil
}

// --- JWT helpers ---

func generateJWT(user *User) (string, error) {
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(config.JWTExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.JWTSecret))
}

// ValidateJWT parses and validates a JWT string, returning the claims.
func ValidateJWT(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(config.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
