package lobby

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrLobbyNotFound = errors.New("lobby not found")
	ErrAlreadyJoined = errors.New("already in this lobby")
	ErrNotInLobby    = errors.New("not in this lobby")
	ErrWrongPassword = errors.New("incorrect lobby password")
	ErrNotHost       = errors.New("only the host can perform this action")
	ErrGameStarted   = errors.New("game has already started")
)

// Store is the in-memory lobby registry.
type Store struct {
	mu      sync.RWMutex
	lobbies map[string]*Lobby
}

func NewStore() *Store {
	return &Store{lobbies: make(map[string]*Lobby)}
}

func (s *Store) Add(l *Lobby) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lobbies[l.Code] = l
}

func (s *Store) Get(code string) (*Lobby, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	l, ok := s.lobbies[code]
	if !ok {
		return nil, ErrLobbyNotFound
	}
	return l, nil
}

func (s *Store) Delete(code string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.lobbies, code)
}

func (s *Store) All() []*Lobby {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Lobby, 0, len(s.lobbies))
	for _, l := range s.lobbies {
		out = append(out, l)
	}
	return out
}

// Service contains the lobby business logic.
type Service struct {
	store *Store
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

func (s *Service) CreateLobby(hostID, hostUsername string, req CreateLobbyRequest) (*Lobby, error) {
	code, err := generateCode()
	if err != nil {
		return nil, err
	}

	l := &Lobby{
		Code:         code,
		Name:         req.Name,
		HostID:       hostID,
		Locked:       req.Password != "",
		Players:      make(map[string]*Player),
		NumComedians: req.NumComedians,
		State:        GameStateWaiting,
	}

	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		l.passwordHash = string(hash)
	}

	l.Players[hostID] = &Player{ID: hostID, Username: hostUsername}
	s.store.Add(l)
	return l, nil
}

func (s *Service) ListLobbies() []LobbyResponse {
	all := s.store.All()
	out := make([]LobbyResponse, 0, len(all))
	for _, l := range all {
		l.RLock()
		out = append(out, toLobbyResponse(l))
		l.RUnlock()
	}
	return out
}

func (s *Service) JoinLobby(playerID, username string, req JoinLobbyRequest) (*Lobby, error) {
	l, err := s.store.Get(req.Code)
	if err != nil {
		return nil, err
	}

	l.Lock()
	defer l.Unlock()

	if l.State != GameStateWaiting {
		return nil, ErrGameStarted
	}
	if _, exists := l.Players[playerID]; exists {
		return nil, ErrAlreadyJoined
	}
	if l.Locked {
		if err := bcrypt.CompareHashAndPassword([]byte(l.passwordHash), []byte(req.Password)); err != nil {
			return nil, ErrWrongPassword
		}
	}

	l.Players[playerID] = &Player{
		ID:           playerID,
		Username:     username,
		TokenBalance: req.TokenBalance,
	}
	return l, nil
}

func (s *Service) LeaveLobby(playerID string, req LeaveLobbyRequest) (closed bool, err error) {
	l, err := s.store.Get(req.Code)
	if err != nil {
		return false, err
	}

	l.Lock()
	defer l.Unlock()

	if _, exists := l.Players[playerID]; !exists {
		return false, ErrNotInLobby
	}

	delete(l.Players, playerID)

	if len(l.Players) == 0 {
		s.store.Delete(req.Code)
		return true, nil
	}

	// Reassign host if necessary.
	if l.HostID == playerID {
		for id := range l.Players {
			l.HostID = id
			break
		}
	}
	return false, nil
}

func (s *Service) GetLobby(code string) (*Lobby, error) {
	return s.store.Get(code)
}

// SetGameState transitions a lobby to a new state (caller must be host).
func (s *Service) SetGameState(callerID, code string, state GameState) error {
	l, err := s.store.Get(code)
	if err != nil {
		return err
	}
	l.Lock()
	defer l.Unlock()

	if l.HostID != callerID {
		return ErrNotHost
	}
	l.State = state
	return nil
}

// ApplyFirstPickAdvantage marks a player as having first-pick and deducts tokens in-memory.
func (s *Service) ApplyFirstPickAdvantage(playerID, lobbyCode string, cost int) (delta int, err error) {
	l, err := s.store.Get(lobbyCode)
	if err != nil {
		return 0, err
	}

	l.Lock()
	defer l.Unlock()

	p, exists := l.Players[playerID]
	if !exists {
		return 0, ErrNotInLobby
	}
	if p.TokenBalance < cost {
		return 0, errors.New("insufficient token balance")
	}

	p.TokenBalance -= cost
	p.HasFirstPick = true
	return -cost, nil
}

// --- helpers ---

func toLobbyResponse(l *Lobby) LobbyResponse {
	return LobbyResponse{
		Code:         l.Code,
		Name:         l.Name,
		HostID:       l.HostID,
		Locked:       l.Locked,
		PlayerCount:  len(l.Players),
		NumComedians: l.NumComedians,
		State:        l.State,
	}
}

func generateCode() (string, error) {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil // 6-character hex code
}
