package networking

import (
	"sync"
	"time"

	"github.com/Parkreiner/bingo"
	"github.com/google/uuid"
)

type uuidRegistryEntry struct {
	lastUsed  time.Time
	ipAddress string
	id        uuid.UUID
}

// uuidRegistry maps each IP address to a different UUID
type uuidRegistry struct {
	entries map[string]uuidRegistryEntry
	mtx     *sync.Mutex
}

func newUUIDRegistry() uuidRegistry {
	return uuidRegistry{
		entries: make(map[string]uuidRegistryEntry),
		mtx:     &sync.Mutex{},
	}
}

func (ur uuidRegistry) upsertAddress(ipAddress string) uuid.UUID {
	ur.mtx.Lock()
	defer ur.mtx.Unlock()

	entry, ok := ur.entries[ipAddress]
	if ok {
		return entry.id
	}

	newID := uuid.New()
	ur.entries[ipAddress] = uuidRegistryEntry{
		id:        newID,
		ipAddress: ipAddress,
		lastUsed:  time.Now(),
	}

	return newID
}

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
