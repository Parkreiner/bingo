package bingo

import (
	"time"

	"github.com/google/uuid"
)

// GameEventType indicates the type and context of a new event's message
// Todo: Definitely need to revamp this type with better granularity
type GameEventType string

const (
	EventTypeUpdate GameEventType = "update"
	EventTypeError  GameEventType = "error"
)

type GameEvent struct {
	ID      uuid.UUID     `json:"id"`
	Phase   GamePhase     `json:"phase"`
	Type    GameEventType `json:"event_type"`
	Created time.Time     `json:"creation_timestamp"`
	Message string        `json:"message"`
	// If an event slice is empty/nil, it's assumed that the event should be
	// broadcast to all players
	RecipientPlayerIDs []string `json:"recipient_player_ids"`
}
