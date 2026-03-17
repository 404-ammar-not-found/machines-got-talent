package auth

import (
	"errors"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/machines-got-talent/backend/pkg/config"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrInvalidPassword = errors.New("invalid password")
	ErrEmailTaken      = errors.New("email already registered")
	ErrUsernameTaken   = errors.New("username already taken")
)

// Service holds an in-memory user store and provides auth business logic.
type Service struct {
	mu      sync.RWMutex
	byEmail map[string]*User
	byID    map[string]*User
}

func NewService() *Service {
	return &Service{
		byEmail: make(map[string]*User),
		byID:    make(map[string]*User),
	}
}

func (s *Service) Register(req RegisterRequest) (*AuthResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.byEmail[req.Email]; exists {
		return nil, ErrEmailTaken
	}
	for _, u := range s.byEmail {
		if u.Username == req.Username {
			return nil, ErrUsernameTaken
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &User{
		ID:           uuid.NewString(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hash),
	}
	s.byEmail[req.Email] = user
	s.byID[user.ID] = user

	token, err := generateJWT(user)
	if err != nil {
		return nil, err
	}
	return &AuthResponse{Token: token, User: *user}, nil
}

func (s *Service) Login(req LoginRequest) (*AuthResponse, error) {
	s.mu.RLock()
	user, exists := s.byEmail[req.Email]
	s.mu.RUnlock()

	if !exists {
		return nil, ErrUserNotFound
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidPassword
	}

	token, err := generateJWT(user)
	if err != nil {
		return nil, err
	}
	return &AuthResponse{Token: token, User: *user}, nil
}

// ResetPassword generates a reset token (email delivery not implemented).
func (s *Service) ResetPassword(req ResetPasswordRequest) (*ResetPasswordResponse, error) {
	s.mu.RLock()
	_, exists := s.byEmail[req.Email]
	s.mu.RUnlock()

	if !exists {
		return nil, ErrUserNotFound
	}

	resetToken := uuid.NewString()
	return &ResetPasswordResponse{
		ResetToken: resetToken,
		Message:    "Use this token to reset your password (email delivery not yet implemented).",
	}, nil
}

func (s *Service) GetUserByID(id string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.byID[id]
	if !exists {
		return nil, ErrUserNotFound
	}
	return user, nil
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
