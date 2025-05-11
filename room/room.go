package room

import (
	"github.com/Parkreiner/bingo"
	"github.com/Parkreiner/bingo/game"
	"github.com/google/uuid"
)

// JoinCode is a four-letter code for joining a game that a host has already
// created
type JoinCode string

type EventType string

const (
	EventTypeGameUpdate EventType = "game_update"
	EventTypeError      EventType = "error"
)

type Event struct {
	ID        uuid.UUID
	EventType EventType
	Message   string
	// If an event slice is empty, it's assumed that the event should be
	// broadcast to all players
	RecipientPlayerIDs []string
}

type Room struct {
	id       uuid.UUID
	joinCode JoinCode
	game     *game.Game
	events   []Event
}

type ClientRoomSnapshot struct {
	id             uuid.UUID
	joinCode       JoinCode
	phase          game.Phase
	winningPlayers []game.Player
	suspension     *game.PlayerSuspension
	cards          []bingo.Card
	events         []Event
}
