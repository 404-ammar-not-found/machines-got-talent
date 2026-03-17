package game

import "encoding/json"

// --- Game phase ---

type Phase string

const (
	PhaseDraft    Phase = "draft"
	PhaseRound    Phase = "round"
	PhaseFinished Phase = "finished"
)

// --- Vote type ---

type VoteType string

const (
	VoteBuzzer       VoteType = "buzzer"
	VoteGoldenBuzzer VoteType = "golden_buzzer"
)

// --- Domain types ---

// AIComedian represents one AI agent competing in the tournament.
type AIComedian struct {
	ID          string `json:"id"`
	Personality string `json:"personality"`
	Score       int    `json:"score"`
	Alive       bool   `json:"alive"`
	ElimRound   int    `json:"elim_round"` // round they were eliminated; 0 = still alive / winner
}

// Matchup is a pair of AI IDs competing in a round.
// If AI2 is empty the comedian in AI1 has a bye.
type Matchup struct {
	AI1 string `json:"ai1"`
	AI2 string `json:"ai2,omitempty"`
}

// Vote records one player's vote in a round.
type Vote struct {
	PlayerID string   `json:"player_id"`
	TargetAI string   `json:"target_ai"`
	VoteType VoteType `json:"vote_type"`
}

// PlayerState tracks per-player game data inside an active game.
type PlayerState struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	TokenBalance int    `json:"token_balance"`
	ClaimedAI    string `json:"claimed_ai"`
	HasFirstPick bool   `json:"has_first_pick"`
	Connected    bool   `json:"connected"`
}

// GameState is the complete, authoritative state of one game.
// Engine functions operate on *GameState without locks — callers must lock.
type GameState struct {
	LobbyCode   string                  `json:"lobby_code"`
	HostID      string                  `json:"host_id"`
	Phase       Phase                   `json:"phase"`
	Round       int                     `json:"round"`
	Players     map[string]*PlayerState `json:"players"`
	DraftOrder  []string                `json:"draft_order"`  // player IDs in pick order
	DraftIndex  int                     `json:"draft_index"`  // index into DraftOrder
	AIComedians map[string]*AIComedian  `json:"ai_comedians"` // keyed by AI ID
	Matchups    []Matchup               `json:"matchups"`     // current round matchups
	Votes       []Vote                  `json:"votes"`        // votes cast this round
	TokenDeltas map[string]int          `json:"token_deltas"` // final rewards
}

// --- WebSocket event names ---

const (
	EventLobbyPlayerJoined = "lobby:player_joined"
	EventLobbyPlayerLeft   = "lobby:player_left"
	EventGameStart         = "game:start"
	EventGameDraftPick     = "game:draft_pick"
	EventGameDraftUpdate   = "game:draft_update"
	EventGameRoundStart    = "game:round_start"
	EventGameVote          = "game:vote"
	EventGameForceEndRound = "game:force_end_round"
	EventGameRoundResult   = "game:round_result"
	EventGameEnd           = "game:end"
	EventError             = "error"
)

// WSMessage is the envelope for every WebSocket message.
type WSMessage struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data,omitempty"`
}

// mustMarshal is a helper used only in constructors; panics on logic errors.
func mustMarshal(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// NewWSMessage builds a WSMessage, marshalling data into the envelope.
func NewWSMessage(event string, data any) WSMessage {
	return WSMessage{Event: event, Data: mustMarshal(data)}
}

// --- Inbound payload types (Client → Server) ---

type DraftPickPayload struct {
	AIID string `json:"ai_id"`
}

type VotePayload struct {
	AIID     string   `json:"ai_id"`
	VoteType VoteType `json:"vote_type"`
}

// --- Outbound payload types (Server → Client) ---

type PlayerJoinedPayload struct {
	PlayerID string `json:"player_id"`
	Username string `json:"username"`
}

type PlayerLeftPayload struct {
	PlayerID string `json:"player_id"`
	Username string `json:"username"`
}

type GameStartPayload struct {
	AIComedians []*AIComedian `json:"ai_comedians"`
	DraftOrder  []string      `json:"draft_order"`
	NextPicker  string        `json:"next_picker"`
}

type DraftUpdatePayload struct {
	Picks      map[string]string `json:"picks"`       // player_id → ai_id
	NextPicker string            `json:"next_picker"` // "" when draft is complete
	Complete   bool              `json:"complete"`
}

type RoundStartPayload struct {
	Round    int       `json:"round"`
	Matchups []Matchup `json:"matchups"`
}

type RoundResultPayload struct {
	Round     int               `json:"round"`
	Eliminated []string         `json:"eliminated"`
	Scores    map[string]int    `json:"scores"`
}

type GameEndPayload struct {
	WinnerAI     string         `json:"winner_ai"`
	WinnerPlayer string         `json:"winner_player"`
	TokenDeltas  map[string]int `json:"token_deltas"`
}
