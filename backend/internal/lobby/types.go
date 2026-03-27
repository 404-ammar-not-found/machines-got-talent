package lobby

import "sync"

// GameState tracks the phase of a lobby's game session.
type GameState string

const (
	GameStateWaiting    GameState = "waiting"
	GameStateDraft      GameState = "draft"
	GameStateInProgress GameState = "in_progress"
	GameStateFinished   GameState = "finished"
)

// Player represents one participant inside a lobby session.
// TokenBalance is received from the frontend — never persisted here.
type Player struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	TokenBalance int    `json:"token_balance"`
	ClaimedAI    string `json:"claimed_ai,omitempty"`
	HasFirstPick bool   `json:"has_first_pick"`
}

// Lobby holds all runtime state for one game room.
type Lobby struct {
	mu sync.RWMutex `json:"-"`

	Code         string             `json:"code"`
	Name         string             `json:"name"`
	HostID       string             `json:"host_id"`
	passwordHash string             // bcrypt hash; empty == no password
	Locked       bool               `json:"locked"`
	Players      map[string]*Player `json:"players"`
	NumComedians int                `json:"num_comedians"`
	State        GameState          `json:"state"`
}

// Lock / RLock expose the embedded mutex for callers that need atomic reads.
func (l *Lobby) Lock()    { l.mu.Lock() }
func (l *Lobby) Unlock()  { l.mu.Unlock() }
func (l *Lobby) RLock()   { l.mu.RLock() }
func (l *Lobby) RUnlock() { l.mu.RUnlock() }

// --- HTTP request / response types ---

type CreateLobbyRequest struct {
	Name         string `json:"name"          binding:"required"`
	Password     string `json:"password"`
	NumComedians int    `json:"num_comedians" binding:"required,min=2,max=16"`
}

type JoinLobbyRequest struct {
	Code         string `json:"code"          binding:"required"`
	Password     string `json:"password"`
	TokenBalance int    `json:"token_balance"`
}

type LeaveLobbyRequest struct {
	Code string `json:"code" binding:"required"`
}

// LobbyResponse is the public view of a lobby (no password data).
type LobbyResponse struct {
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	HostID       string    `json:"host_id"`
	Locked       bool      `json:"locked"`
	PlayerCount  int       `json:"player_count"`
	NumComedians int       `json:"num_comedians"`
	State        GameState `json:"state"`
}
