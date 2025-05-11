package networking

import (
	"github.com/Parkreiner/bingo"
	"github.com/google/uuid"
)

// JoinCode is a four-letter code for joining a game that a host has already
// created
type JoinCode string

// GameEventType indicates the type and context of a new event's message
type GameEventType string

const (
	EventTypeGameUpdate GameEventType = "game_update"
	EventTypeError      GameEventType = "error"
)

type GameEvent struct {
	ID        uuid.UUID     `json:"id"`
	EventType GameEventType `json:"event_type"`
	Message   string        `json:"message"`
	// If an event slice is empty, it's assumed that the event should be
	// broadcast to all players
	RecipientPlayerIDs []string `json:"recipient_player_ids"`
}

// Room represents a stateful room of players who are all playing the same
// game of bingo.
type Room struct {
	id       uuid.UUID
	joinCode JoinCode
	game     bingo.Game
	events   []GameEvent
}

type clientRoomSnapshot struct {
	ID             uuid.UUID               `json:"id"`
	JoinCode       JoinCode                `json:"join_code"`
	Phase          bingo.GamePhase         `json:"phase"`
	PlayerCount    int                     `json:"player_count"`
	WinningPlayers []bingo.User            `json:"winning_players"`
	Suspension     *bingo.PlayerSuspension `json:"suspension"`
	Cards          []bingo.Card            `json:"cards"`
	Events         []GameEvent             `json:"events"`
}
