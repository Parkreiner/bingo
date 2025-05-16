package server

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

type playerSessionSnapshot struct {
	RoomID         uuid.UUID               `json:"roomId"`
	PlayerID       uuid.UUID               `json:"playerId"`
	JoinCode       JoinCode                `json:"joinCode"`
	Phase          bingo.GamePhase         `json:"phase"`
	PlayerCount    int                     `json:"playerCount"`
	WinningPlayers []bingo.Player          `json:"winningPlayers"`
	Suspension     *bingo.PlayerSuspension `json:"suspension"`
	Cards          []bingo.Card            `json:"cards"`
	Events         []bingo.GameEvent       `json:"events"`
}
