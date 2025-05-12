package networking

import (
	"github.com/Parkreiner/bingo"
	"github.com/google/uuid"
)

// JoinCode is a four-letter code for joining a game that a host has already
// created
type JoinCode string

// Room represents a stateful room of players who are all playing the same
// game of bingo.
type Room struct {
	id       uuid.UUID
	joinCode JoinCode
	game     bingo.GameManager
	events   []bingo.GameEvent
}

type clientRoomSnapshot struct {
	ID             uuid.UUID               `json:"id"`
	JoinCode       JoinCode                `json:"join_code"`
	Phase          bingo.GamePhase         `json:"phase"`
	PlayerCount    int                     `json:"player_count"`
	WinningPlayers []bingo.Player          `json:"winning_players"`
	Suspension     *bingo.PlayerSuspension `json:"suspension"`
	Cards          []bingo.Card            `json:"cards"`
	Events         []bingo.GameEvent       `json:"events"`
}
