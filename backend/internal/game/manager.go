package game

import (
	"encoding/json"
	"errors"
	"log"
	"math"
	"sync"

	"github.com/machines-got-talent/backend/internal/ai"
	"github.com/machines-got-talent/backend/internal/lobby"
)

var ErrGameNotFound = errors.New("game not found for lobby")

// Manager coordinates the game engine and the WebSocket hub for one lobby.
type Manager struct {
	mu    sync.Mutex
	state *GameState
	hub   *Hub
}

func newManager(state *GameState) *Manager {
	m := &Manager{
		state: state,
		hub:   newHub(),
	}
	m.hub.onMessage = m.handleClientMessage
	m.hub.onLeave = m.handlePlayerLeft
	go m.hub.Run()
	return m
}

// Store is the global registry of active game managers keyed by lobby code.
type Store struct {
	mu       sync.RWMutex
	managers map[string]*Manager
}

func NewStore() *Store {
	return &Store{managers: make(map[string]*Manager)}
}

func (s *Store) Get(lobbyCode string) (*Manager, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.managers[lobbyCode]
	if !ok {
		return nil, ErrGameNotFound
	}
	return m, nil
}

func (s *Store) set(lobbyCode string, m *Manager) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.managers[lobbyCode] = m
}

func (s *Store) delete(lobbyCode string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.managers, lobbyCode)
}

// StartGame creates a game manager for the lobby, calls the AI service, and
// broadcasts the game:start event. Callers must ensure the lobby is in the
// waiting state before calling.
func (s *Store) StartGame(l *lobby.Lobby, aiClient *ai.Client) (*Manager, error) {
	l.RLock()
	lobbyCode := l.Code
	hostID := l.HostID
	numComedians := l.NumComedians
	players := snapshotPlayers(l)
	l.RUnlock()

	comedians, err := aiClient.CreateAgents(numComedians)
	if err != nil {
		return nil, err
	}

	state := NewGameState(lobbyCode, hostID, players, comedians)
	m := newManager(state)
	s.set(lobbyCode, m)

	// Broadcast game:start to all connected WebSocket clients.
	aiList := make([]*AIComedian, 0, len(state.AIComedians))
	for _, ai := range state.AIComedians {
		aiList = append(aiList, ai)
	}
	nextPicker := ""
	if len(state.DraftOrder) > 0 {
		nextPicker = state.DraftOrder[0]
	}

	m.hub.Broadcast(NewWSMessage(EventGameStart, GameStartPayload{
		AIComedians: aiList,
		DraftOrder:  state.DraftOrder,
		NextPicker:  nextPicker,
	}))

	return m, nil
}

// ConnectClient upgrades an existing HTTP connection to WebSocket and registers
// it with the lobby's hub.
func (s *Store) ConnectClient(lobbyCode string, c *Client) error {
	m, err := s.Get(lobbyCode)
	if err != nil {
		return err
	}

	m.mu.Lock()
	if p, ok := m.state.Players[c.PlayerID]; ok {
		p.Connected = true
	}
	m.mu.Unlock()

	m.hub.register <- c

	// Announce join to others.
	m.hub.Broadcast(NewWSMessage(EventLobbyPlayerJoined, PlayerJoinedPayload{
		PlayerID: c.PlayerID,
		Username: c.Username,
	}))

	go c.WritePump()
	go c.ReadPump()
	return nil
}

// --- inbound message router ---

func (m *Manager) handleClientMessage(c *Client, msg WSMessage) {
	switch msg.Event {
	case EventGameDraftPick:
		var p DraftPickPayload
		if err := json.Unmarshal(msg.Data, &p); err != nil {
			m.sendError(c, "invalid draft_pick payload")
			return
		}
		m.processDraftPick(c, p)

	case EventGameVote:
		var p VotePayload
		if err := json.Unmarshal(msg.Data, &p); err != nil {
			m.sendError(c, "invalid vote payload")
			return
		}
		m.processVote(c, p)

	case EventGameForceEndRound:
		m.processForceEndRound(c)

	default:
		m.sendError(c, "unknown event: "+msg.Event)
	}
}

// --- game action handlers ---

func (m *Manager) processDraftPick(c *Client, p DraftPickPayload) {
	m.mu.Lock()
	update, err := ProcessDraftPick(m.state, c.PlayerID, p.AIID)
	m.mu.Unlock()

	if err != nil {
		m.sendError(c, err.Error())
		return
	}

	m.hub.Broadcast(NewWSMessage(EventGameDraftUpdate, update))

	if update.Complete {
		m.beginRound()
	}
}

func (m *Manager) processVote(c *Client, p VotePayload) {
	m.mu.Lock()
	err := ProcessVote(m.state, c.PlayerID, p.AIID, p.VoteType)
	allVoted := AllConnectedPlayersVoted(m.state)
	m.mu.Unlock()

	if err != nil {
		m.sendError(c, err.Error())
		return
	}

	if allVoted {
		m.endRound()
	}
}

func (m *Manager) processForceEndRound(c *Client) {
	m.mu.Lock()
	if m.state.HostID != c.PlayerID {
		m.mu.Unlock()
		m.sendError(c, ErrNotHost.Error())
		return
	}
	m.mu.Unlock()

	m.endRound()
}

// beginRound starts a new tournament round.
func (m *Manager) beginRound() {
	m.mu.Lock()
	payload, err := StartRound(m.state)
	m.mu.Unlock()

	if err != nil {
		log.Printf("StartRound error lobby=%s: %v", m.state.LobbyCode, err)
		return
	}
	m.hub.Broadcast(NewWSMessage(EventGameRoundStart, payload))
}

// endRound resolves the current round, eliminates losers, and either starts
// the next round or ends the game.
func (m *Manager) endRound() {
	m.mu.Lock()
	result, err := EndRound(m.state)
	if err != nil {
		m.mu.Unlock()
		log.Printf("EndRound error lobby=%s: %v", m.state.LobbyCode, err)
		return
	}

	winnerAI, over := CheckGameOver(m.state)
	var deltas map[string]int
	var winnerPlayer string
	if over {
		totalRounds := m.state.Round
		deltas = CalculateRewards(m.state, totalRounds)
		m.state.Phase = PhaseFinished
		// Find the player who claimed the winning AI.
		for id, p := range m.state.Players {
			if p.ClaimedAI == winnerAI {
				winnerPlayer = id
				break
			}
		}
	}
	m.mu.Unlock()

	m.hub.Broadcast(NewWSMessage(EventGameRoundResult, result))

	if over {
		m.hub.Broadcast(NewWSMessage(EventGameEnd, GameEndPayload{
			WinnerAI:     winnerAI,
			WinnerPlayer: winnerPlayer,
			TokenDeltas:  deltas,
		}))
		return
	}

	m.beginRound()
}

// handlePlayerLeft updates connected status and notifies lobby.
func (m *Manager) handlePlayerLeft(playerID string) {
	m.mu.Lock()
	username := ""
	if p, ok := m.state.Players[playerID]; ok {
		p.Connected = false
		username = p.Username
	}
	m.mu.Unlock()

	m.hub.Broadcast(NewWSMessage(EventLobbyPlayerLeft, PlayerLeftPayload{
		PlayerID: playerID,
		Username: username,
	}))
}

func (m *Manager) sendError(c *Client, msg string) {
	m.hub.Send(c.PlayerID, NewWSMessage(EventError, map[string]string{"message": msg}))
}

// --- helpers ---

// snapshotPlayers copies the lobby's player map into PlayerState objects.
func snapshotPlayers(l *lobby.Lobby) map[string]*PlayerState {
	out := make(map[string]*PlayerState, len(l.Players))
	for id, p := range l.Players {
		out[id] = &PlayerState{
			ID:           p.ID,
			Username:     p.Username,
			TokenBalance: p.TokenBalance,
			HasFirstPick: p.HasFirstPick,
			Connected:    false,
		}
	}
	return out
}

// totalRoundsEstimate returns ceil(log2(n)) — used as a denominator for rewards.
// Exported so handler can use it without duplicating logic.
func totalRoundsEstimate(n int) int {
	if n <= 1 {
		return 1
	}
	return int(math.Ceil(math.Log2(float64(n))))
}
