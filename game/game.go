package game

import (
	"github.com/Parkreiner/bingo/ballregistry"
	"github.com/Parkreiner/bingo/cardregistry"
	"github.com/google/uuid"
)

const maxPlayerCapacity = 50

type Phase string

const (
	PhaseGameStart       Phase = "game_start"
	PhaseRoundStart      Phase = "round_start"
	PhaseCalling         Phase = "calling"
	PhaseConfirmingBingo Phase = "confirming_bingo"
	PhaseRoundEnd        Phase = "round_end"
	PhaseRoundTransition Phase = "round_transition"
	PhaseGameOver        Phase = "game_over"
)

type Player struct {
	ID   uuid.UUID
	Name string
}

type PlayerSuspension struct {
	playerID     uuid.UUID
	duration     int
	currentRound int
}

type Game struct {
	ID                uuid.UUID
	maxRounds         int
	Phase             Phase
	ballRegistry      *ballregistry.Registry
	cardRegistry      *cardregistry.Registry
	host              Player
	activePlayers     []Player
	waitlistedPlayers []Player

	// This value keeps track of which player was responsible for winning a
	// given round. The whole player is stored because it's possible for a
	// player to leave the game, so there's no guarantee that an ID in
	// winningPlayers would match with the players field. This field is also
	// used to derive the current round number
	winningPlayers       []Player
	bingoCallerPlayerIDs []uuid.UUID
	suspensions          []PlayerSuspension
	bannedPlayerIDs      []uuid.UUID
}
