package game

import (
	"testing"

	"github.com/machines-got-talent/backend/internal/ai"
	"github.com/machines-got-talent/backend/pkg/config"
)

// --- helpers ---

func makeComedians(ids ...string) []ai.Comedian {
	out := make([]ai.Comedian, len(ids))
	for i, id := range ids {
		out[i] = ai.Comedian{ID: id, Personality: "test"}
	}
	return out
}

func makePlayers(ids ...string) map[string]*PlayerState {
	m := make(map[string]*PlayerState, len(ids))
	for _, id := range ids {
		m[id] = &PlayerState{ID: id, Username: id, Connected: true}
	}
	return m
}

// --- NewGameState ---

func TestNewGameState_InitialValues(t *testing.T) {
	state := NewGameState("lobby1", "p1",
		makePlayers("p1", "p2"),
		makeComedians("ai1", "ai2", "ai3"),
	)

	if state.Phase != PhaseDraft {
		t.Fatalf("expected PhaseDraft, got %s", state.Phase)
	}
	if len(state.AIComedians) != 3 {
		t.Fatalf("expected 3 AI comedians, got %d", len(state.AIComedians))
	}
	for _, ai := range state.AIComedians {
		if ai.Score != config.AIStartingScore {
			t.Errorf("AI %s starting score %d != %d", ai.ID, ai.Score, config.AIStartingScore)
		}
		if !ai.Alive {
			t.Errorf("AI %s should be alive at start", ai.ID)
		}
	}
	if len(state.DraftOrder) != 2 {
		t.Fatalf("expected draft order length 2, got %d", len(state.DraftOrder))
	}
}

func TestNewGameState_FirstPickAtFront(t *testing.T) {
	players := makePlayers("p1", "p2", "p3")
	players["p2"].HasFirstPick = true

	state := NewGameState("lobby1", "p1", players, makeComedians("ai1", "ai2"))

	if state.DraftOrder[0] != "p2" {
		t.Errorf("first pick player should be at index 0, got %s", state.DraftOrder[0])
	}
}

// --- ProcessDraftPick ---

func TestProcessDraftPick_Success(t *testing.T) {
	state := NewGameState("lobby1", "p1",
		makePlayers("p1", "p2"),
		makeComedians("ai1", "ai2"),
	)
	firstPicker := state.DraftOrder[0]

	update, err := ProcessDraftPick(state, firstPicker, "ai1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if update.Picks[firstPicker] != "ai1" {
		t.Errorf("expected pick recorded, got %v", update.Picks)
	}
	if state.DraftIndex != 1 {
		t.Errorf("expected draft index 1, got %d", state.DraftIndex)
	}
}

func TestProcessDraftPick_WrongTurn(t *testing.T) {
	state := NewGameState("lobby1", "p1",
		makePlayers("p1", "p2"),
		makeComedians("ai1", "ai2"),
	)
	wrongPicker := state.DraftOrder[1]

	_, err := ProcessDraftPick(state, wrongPicker, "ai1")
	if err != ErrNotYourTurn {
		t.Errorf("expected ErrNotYourTurn, got %v", err)
	}
}

func TestProcessDraftPick_AlreadyClaimed(t *testing.T) {
	state := NewGameState("lobby1", "p1",
		makePlayers("p1", "p2"),
		makeComedians("ai1", "ai2"),
	)
	first := state.DraftOrder[0]

	ProcessDraftPick(state, first, "ai1") // claim ai1

	second := state.DraftOrder[1] // next player in draft order
	_, err := ProcessDraftPick(state, second, "ai1")
	if err != ErrAIAlreadyClaimed {
		t.Errorf("expected ErrAIAlreadyClaimed, got %v", err)
	}
}

func TestProcessDraftPick_DraftCompletesAfterLastPick(t *testing.T) {
	players := makePlayers("p1", "p2")
	state := NewGameState("lobby1", "p1", players, makeComedians("ai1", "ai2"))

	first := state.DraftOrder[0]
	second := state.DraftOrder[1]

	update, _ := ProcessDraftPick(state, first, "ai1")
	if update.Complete {
		t.Error("draft should not be complete after first pick")
	}

	update, _ = ProcessDraftPick(state, second, "ai2")
	if !update.Complete {
		t.Error("draft should be complete after last pick")
	}
	if state.Phase != PhaseRound {
		t.Errorf("expected PhaseRound after draft, got %s", state.Phase)
	}
}

// --- StartRound ---

func TestStartRound_BuildsMatchups(t *testing.T) {
	state := NewGameState("lobby1", "p1",
		makePlayers("p1", "p2"),
		makeComedians("ai1", "ai2", "ai3", "ai4"),
	)
	state.Phase = PhaseRound

	payload, err := StartRound(state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Round != 1 {
		t.Errorf("expected round 1, got %d", state.Round)
	}
	if len(payload.Matchups) != 2 {
		t.Errorf("expected 2 matchups for 4 AIs, got %d", len(payload.Matchups))
	}
}

func TestStartRound_OddNumberGetsBye(t *testing.T) {
	state := NewGameState("lobby1", "p1",
		makePlayers("p1"),
		makeComedians("ai1", "ai2", "ai3"),
	)
	state.Phase = PhaseRound

	payload, _ := StartRound(state)
	byeCount := 0
	for _, m := range payload.Matchups {
		if m.AI2 == "" {
			byeCount++
		}
	}
	if byeCount != 1 {
		t.Errorf("expected 1 bye matchup for 3 AIs, got %d", byeCount)
	}
}

// --- ProcessVote ---

func TestProcessVote_Buzzer(t *testing.T) {
	state := NewGameState("lobby1", "p1",
		makePlayers("p1"),
		makeComedians("ai1", "ai2"),
	)
	state.Phase = PhaseRound
	StartRound(state)

	initialScore := state.AIComedians["ai1"].Score
	err := ProcessVote(state, "p1", "ai1", VoteBuzzer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.AIComedians["ai1"].Score != initialScore+config.BuzzerPoints {
		t.Errorf("buzzer did not apply correctly")
	}
}

func TestProcessVote_GoldenBuzzer(t *testing.T) {
	state := NewGameState("lobby1", "p1",
		makePlayers("p1"),
		makeComedians("ai1", "ai2"),
	)
	state.Phase = PhaseRound
	StartRound(state)

	initialScore := state.AIComedians["ai1"].Score
	err := ProcessVote(state, "p1", "ai1", VoteGoldenBuzzer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.AIComedians["ai1"].Score != initialScore+config.GoldenBuzzerPoints {
		t.Errorf("golden buzzer did not apply correctly")
	}
}

func TestProcessVote_DoubleVotePrevented(t *testing.T) {
	state := NewGameState("lobby1", "p1",
		makePlayers("p1"),
		makeComedians("ai1", "ai2"),
	)
	state.Phase = PhaseRound
	StartRound(state)

	ProcessVote(state, "p1", "ai1", VoteBuzzer)
	err := ProcessVote(state, "p1", "ai2", VoteBuzzer)
	if err != ErrAlreadyVoted {
		t.Errorf("expected ErrAlreadyVoted, got %v", err)
	}
}

func TestProcessVote_CannotVoteOwnAI(t *testing.T) {
	state := NewGameState("lobby1", "p1",
		makePlayers("p1"),
		makeComedians("ai1", "ai2"),
	)
	state.Players["p1"].ClaimedAI = "ai1"
	state.Phase = PhaseRound
	StartRound(state)

	err := ProcessVote(state, "p1", "ai1", VoteBuzzer)
	if err != ErrCannotVoteOwnAI {
		t.Errorf("expected ErrCannotVoteOwnAI, got %v", err)
	}
}

// --- EndRound ---

func TestEndRound_EliminatesLowerScore(t *testing.T) {
	state := NewGameState("lobby1", "p1",
		makePlayers("p1", "p2"),
		makeComedians("ai1", "ai2"),
	)
	state.Phase = PhaseRound
	StartRound(state)

	// Force ai1 to have lower score.
	state.AIComedians["ai1"].Score = 1
	state.AIComedians["ai2"].Score = 5

	// Ensure matchup contains ai1 vs ai2.
	state.Matchups = []Matchup{{AI1: "ai1", AI2: "ai2"}}

	result, err := EndRound(state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Eliminated) != 1 || result.Eliminated[0] != "ai1" {
		t.Errorf("expected ai1 eliminated, got %v", result.Eliminated)
	}
	if state.AIComedians["ai1"].Alive {
		t.Error("ai1 should be marked as not alive")
	}
}

// --- CheckGameOver ---

func TestCheckGameOver_ReturnsTrueOnOneAlive(t *testing.T) {
	state := NewGameState("lobby1", "p1",
		makePlayers("p1"),
		makeComedians("ai1", "ai2"),
	)
	state.Phase = PhaseRound
	state.AIComedians["ai2"].Alive = false

	winner, over := CheckGameOver(state)
	if !over {
		t.Error("expected game to be over")
	}
	if winner != "ai1" {
		t.Errorf("expected winner ai1, got %s", winner)
	}
}

func TestCheckGameOver_FalseOnMultipleAlive(t *testing.T) {
	state := NewGameState("lobby1", "p1",
		makePlayers("p1"),
		makeComedians("ai1", "ai2"),
	)
	state.Phase = PhaseRound

	_, over := CheckGameOver(state)
	if over {
		t.Error("game should not be over with 2 AIs alive")
	}
}

// --- CalculateRewards ---

func TestCalculateRewards_WinnerGetsTopReward(t *testing.T) {
	state := NewGameState("lobby1", "p1",
		makePlayers("p1", "p2"),
		makeComedians("ai1", "ai2"),
	)
	state.Players["p1"].ClaimedAI = "ai1"
	state.Players["p2"].ClaimedAI = "ai2"
	state.AIComedians["ai2"].Alive = false
	state.AIComedians["ai2"].ElimRound = 1

	deltas := CalculateRewards(state, 2)
	if deltas["p1"] != config.WinnerReward {
		t.Errorf("winner should get %d tokens, got %d", config.WinnerReward, deltas["p1"])
	}
	if deltas["p2"] <= 0 {
		t.Error("loser should still get participation/survival reward")
	}
}

func TestCalculateRewards_NoClaimedAIGetsParticipation(t *testing.T) {
	state := NewGameState("lobby1", "p1",
		makePlayers("p1"),
		makeComedians("ai1"),
	)
	// p1 never claimed an AI
	state.Players["p1"].ClaimedAI = ""

	deltas := CalculateRewards(state, 1)
	if deltas["p1"] != config.ParticipationReward {
		t.Errorf("expected participation reward %d, got %d", config.ParticipationReward, deltas["p1"])
	}
}
