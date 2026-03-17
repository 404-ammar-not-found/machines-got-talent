// Package game contains all pure game-logic functions. None of these functions
// touch networking or locking — callers are responsible for synchronisation.
package game

import (
	"errors"
	"math"
	"math/rand"

	"github.com/machines-got-talent/backend/internal/ai"
	"github.com/machines-got-talent/backend/pkg/config"
)

var (
	ErrDraftComplete     = errors.New("draft is already complete")
	ErrNotYourTurn       = errors.New("it is not your turn to pick")
	ErrAIAlreadyClaimed  = errors.New("that AI comedian has already been claimed")
	ErrAINotFound        = errors.New("ai comedian not found")
	ErrAlreadyVoted      = errors.New("you have already voted this round")
	ErrCannotVoteOwnAI   = errors.New("you cannot vote on your own AI comedian")
	ErrAIEliminated      = errors.New("that AI comedian has already been eliminated")
	ErrWrongPhase        = errors.New("action not allowed in the current game phase")
	ErrNotHost           = errors.New("only the host can perform this action")
)

// NewGameState builds the initial GameState from lobby data and AI comedian list.
// DraftOrder is randomised, with the first-pick player placed first if present.
func NewGameState(lobbyCode, hostID string, players map[string]*PlayerState, comedians []ai.Comedian) *GameState {
	aiMap := make(map[string]*AIComedian, len(comedians))
	for _, c := range comedians {
		aiMap[c.ID] = &AIComedian{
			ID:          c.ID,
			Personality: c.Personality,
			Score:       config.AIStartingScore,
			Alive:       true,
		}
	}

	// Build draft order: randomise all players, then move first-pick player to front.
	order := make([]string, 0, len(players))
	var firstPickPlayer string
	for id, p := range players {
		if p.HasFirstPick {
			firstPickPlayer = id
		} else {
			order = append(order, id)
		}
	}
	rand.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
	if firstPickPlayer != "" {
		order = append([]string{firstPickPlayer}, order...)
	}

	return &GameState{
		LobbyCode:   lobbyCode,
		HostID:      hostID,
		Phase:       PhaseDraft,
		Round:       0,
		Players:     players,
		DraftOrder:  order,
		DraftIndex:  0,
		AIComedians: aiMap,
		Votes:       nil,
		TokenDeltas: make(map[string]int),
	}
}

// ProcessDraftPick claims an AI comedian for the current picker.
// Returns the updated state and a DraftUpdatePayload to broadcast.
func ProcessDraftPick(state *GameState, playerID, aiID string) (*DraftUpdatePayload, error) {
	if state.Phase != PhaseDraft {
		return nil, ErrWrongPhase
	}
	if state.DraftIndex >= len(state.DraftOrder) {
		return nil, ErrDraftComplete
	}
	if state.DraftOrder[state.DraftIndex] != playerID {
		return nil, ErrNotYourTurn
	}

	ai, ok := state.AIComedians[aiID]
	if !ok {
		return nil, ErrAINotFound
	}
	// Check if already claimed.
	for _, p := range state.Players {
		if p.ClaimedAI == aiID {
			return nil, ErrAIAlreadyClaimed
		}
	}

	state.Players[playerID].ClaimedAI = aiID
	ai.Personality = ai.Personality // no-op; personality already set
	state.DraftIndex++

	picks := currentPicks(state)
	nextPicker := ""
	complete := state.DraftIndex >= len(state.DraftOrder)
	if !complete {
		nextPicker = state.DraftOrder[state.DraftIndex]
	} else {
		state.Phase = PhaseRound // draft done, transition to tournament
	}

	return &DraftUpdatePayload{
		Picks:      picks,
		NextPicker: nextPicker,
		Complete:   complete,
	}, nil
}

// StartRound advances the round counter, builds matchups from alive AIs, and
// resets the vote list. Returns the RoundStartPayload to broadcast.
func StartRound(state *GameState) (*RoundStartPayload, error) {
	if state.Phase != PhaseRound {
		return nil, ErrWrongPhase
	}

	alive := aliveAIs(state)
	if len(alive) < 2 {
		return nil, errors.New("not enough AIs to start a round")
	}

	state.Round++
	state.Votes = nil

	// Shuffle and pair up.
	rand.Shuffle(len(alive), func(i, j int) { alive[i], alive[j] = alive[j], alive[i] })
	matchups := make([]Matchup, 0, (len(alive)+1)/2)
	for i := 0; i+1 < len(alive); i += 2 {
		matchups = append(matchups, Matchup{AI1: alive[i], AI2: alive[i+1]})
	}
	if len(alive)%2 == 1 {
		// Odd AI out gets a bye.
		matchups = append(matchups, Matchup{AI1: alive[len(alive)-1]})
	}
	state.Matchups = matchups

	return &RoundStartPayload{Round: state.Round, Matchups: matchups}, nil
}

// ProcessVote records a player's vote, validating rules.
func ProcessVote(state *GameState, playerID, targetAIID string, vt VoteType) error {
	if state.Phase != PhaseRound {
		return ErrWrongPhase
	}

	ai, ok := state.AIComedians[targetAIID]
	if !ok {
		return ErrAINotFound
	}
	if !ai.Alive {
		return ErrAIEliminated
	}

	// Players cannot vote on their own AI.
	p := state.Players[playerID]
	if p != nil && p.ClaimedAI == targetAIID {
		return ErrCannotVoteOwnAI
	}

	// One vote per player per round.
	for _, v := range state.Votes {
		if v.PlayerID == playerID {
			return ErrAlreadyVoted
		}
	}

	points := config.BuzzerPoints
	if vt == VoteGoldenBuzzer {
		points = config.GoldenBuzzerPoints
	}
	ai.Score += points

	state.Votes = append(state.Votes, Vote{
		PlayerID: playerID,
		TargetAI: targetAIID,
		VoteType: vt,
	})
	return nil
}

// AllConnectedPlayersVoted returns true when every online player has cast a vote this round.
func AllConnectedPlayersVoted(state *GameState) bool {
	voters := make(map[string]bool, len(state.Votes))
	for _, v := range state.Votes {
		voters[v.PlayerID] = true
	}
	for id, p := range state.Players {
		if p.Connected && !voters[id] {
			return false
		}
	}
	return true
}

// EndRound resolves each matchup: the AI with fewer points is eliminated.
// Ties are broken randomly. Returns a RoundResultPayload to broadcast.
func EndRound(state *GameState) (*RoundResultPayload, error) {
	if state.Phase != PhaseRound {
		return nil, ErrWrongPhase
	}

	eliminated := []string{}
	scores := currentScores(state)

	for _, m := range state.Matchups {
		if m.AI2 == "" {
			// Bye — no elimination.
			continue
		}
		ai1 := state.AIComedians[m.AI1]
		ai2 := state.AIComedians[m.AI2]

		loser := determineLoser(ai1, ai2)
		loser.Alive = false
		loser.ElimRound = state.Round
		eliminated = append(eliminated, loser.ID)
	}

	return &RoundResultPayload{
		Round:      state.Round,
		Eliminated: eliminated,
		Scores:     scores,
	}, nil
}

// CheckGameOver returns (winnerAIID, true) if only one AI remains alive.
func CheckGameOver(state *GameState) (string, bool) {
	alive := aliveAIs(state)
	if len(alive) == 1 {
		return alive[0], true
	}
	return "", false
}

// CalculateRewards assigns token rewards to players based on how far their AI progressed.
// The result is stored in state.TokenDeltas and also returned for broadcasting.
func CalculateRewards(state *GameState, totalRounds int) map[string]int {
	for id, p := range state.Players {
		if p.ClaimedAI == "" {
			state.TokenDeltas[id] = config.ParticipationReward
			continue
		}

		ai := state.AIComedians[p.ClaimedAI]
		if !ai.Alive {
			// Survived until round elim_round; scale reward.
			survivalFraction := float64(ai.ElimRound) / float64(totalRounds)
			reward := int(math.Round(survivalFraction * float64(config.SurvivorRewardMax)))
			if reward < config.ParticipationReward {
				reward = config.ParticipationReward
			}
			state.TokenDeltas[id] = reward
		} else {
			// AI is still alive — this player is the winner.
			state.TokenDeltas[id] = config.WinnerReward
		}
	}
	return state.TokenDeltas
}

// --- internal helpers ---

func aliveAIs(state *GameState) []string {
	out := make([]string, 0, len(state.AIComedians))
	for id, ai := range state.AIComedians {
		if ai.Alive {
			out = append(out, id)
		}
	}
	return out
}

func currentPicks(state *GameState) map[string]string {
	m := make(map[string]string)
	for id, p := range state.Players {
		if p.ClaimedAI != "" {
			m[id] = p.ClaimedAI
		}
	}
	return m
}

func currentScores(state *GameState) map[string]int {
	m := make(map[string]int, len(state.AIComedians))
	for id, ai := range state.AIComedians {
		m[id] = ai.Score
	}
	return m
}

func determineLoser(a1, a2 *AIComedian) *AIComedian {
	if a1.Score < a2.Score {
		return a1
	}
	if a2.Score < a1.Score {
		return a2
	}
	// Tie — random.
	if rand.Intn(2) == 0 {
		return a1
	}
	return a2
}
