package bingo

import (
	"time"

	"github.com/google/uuid"
)

// GameEventType indicates the type and context of a new event's message
// TODO: Definitely need to revamp this type with better granularity
type GameEventType string

// TODO: Add comments for each event type once I figure out exactly how I want
// to structure event types in general
const (
	EventTypeUpdate GameEventType = "update"
	EventTypeError  GameEventType = "error"
)

// GameEvent represents something that has happened in the game (either the
// result of an automatic game update, or a player action).
type GameEvent struct {
	ID          uuid.UUID     `json:"id"`
	CreatedByID uuid.UUID     `json:"createdById"`
	Phase       GamePhase     `json:"phase"`
	Type        GameEventType `json:"event_type"`
	Created     time.Time     `json:"creation_timestamp"`
	Message     string        `json:"message"`
	// If the player ID slice is empty/nil, it's assumed that the event should
	// be broadcast to all players
	RecipientPlayerIDs []uuid.UUID `json:"recipient_player_ids"`
}
