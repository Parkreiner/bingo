// Package bingo contains the main domain logic for playing a bingo game.
package bingo

import (
	"fmt"

	"github.com/google/uuid"
)

const (
	// MinCards represents the minimum number of cards a player is allowed to
	// have in a game.
	MinCards = 1

	// MaxCards represents the maximum number of cards a player is allowed to
	// have in a game.
	MaxCards = 6

	// MaxBallValue is the highest bingo ball value that you can possibly have
	// in an American game of bingo
	MaxBallValue = 75
)

// Ball represents a single number from 1 to 75 (both inclusive) that can
// called during a bingo game
type Ball byte

// FreeSpace represents the space given for free to all players. It is the zero
// value of Ball. It should not be daubed automatically on a bingo card, just so
// that players have more to do in a round
const FreeSpace = Ball(0)

// ParseBall takes any arbitrary int, and attempts to turn it into a bingo ball.
// Will error if the provided value is below 1 or 75.
func ParseBall(rawBallValue int) (Ball, error) {
	if rawBallValue > MaxBallValue {
		return FreeSpace, fmt.Errorf("value %d is not allowed to exceed %d", rawBallValue, MaxBallValue)
	}
	if rawBallValue <= 0 {
		return FreeSpace, fmt.Errorf("value %d is not allowed to fall below 0", rawBallValue)
	}
	return Ball(rawBallValue), nil
}

// Cell represents a single stateful cell on a bingo card.
type Cell struct {
	// The numeric value of a cell. It is assumed that once a full card has been
	// created, Number will remain 100% static for as long as the card remains
	// active in the game
	Number Ball
	// Indicates whether a player has marked the cell with a virtual dauber.
	// This value is allowed to be mutated.
	Daubed bool
}

// Card represents a single stateful card, currently being used by a player.
type Card struct {
	// A 5x5 grid of Bingo cells. Each column corresponds to a different "letter
	// "group" in the bingo board. That is:
	//
	// 1. Column 1 is column B and can have numbers 1–15
	// 2. Column 2 is column I and can have numbers 16–30
	// 3. Column 3 is column N and can have numbers 31–45, along with the free
	//   space in the middle
	// 4. Column 4 is column G and can have numbers 46–60
	// 5. Column 5 is column O and can have numbers 61–75
	//
	// The free space is represented as 0.
	Cells [][]*Cell

	ID       uuid.UUID
	PlayerID uuid.UUID
}

// GenerateBingoBallsForRange creates a range of bingo balls for a given
// contiguous range. If the start or end bounds are invalid, the function will
// return a nil slice instead.
func GenerateBingoBallsForRange(start int, end int) []Ball {
	var cells []Ball
	inputIsInvalid := end <= start ||
		start <= 0 || end <= 0 ||
		start > MaxBallValue || end > MaxBallValue
	if inputIsInvalid {
		return cells
	}

	for i := start; i <= end; i++ {
		cells = append(cells, Ball(i))
	}
	return cells
}

type BallRegistry interface {
	NextAutomaticCall() (Ball, error)
	SyncManualCall(ball Ball) error
	Reset()
}

type CardRegistry interface {
	Start() (terminate func(), err error)
	CheckOutCard(playerID uuid.UUID) (*Card, error)
	ReturnCard(cardID uuid.UUID) error
}

const maxPlayerCapacity = 50

type GamePhase string

const (
	GamePhaseGameStart       GamePhase = "game_start"
	GamePhaseRoundStart      GamePhase = "round_start"
	GamePhaseCalling         GamePhase = "calling"
	GamePhaseConfirmingBingo GamePhase = "confirming_bingo"
	GamePhaseRoundEnd        GamePhase = "round_end"
	GamePhaseRoundTransition GamePhase = "round_transition"
	GamePhaseGameOver        GamePhase = "game_over"
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
	Phase             GamePhase
	ballRegistry      BallRegistry
	cardRegistry      CardRegistry
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
	game     *Game
	events   []Event
}

type ClientRoomSnapshot struct {
	id             uuid.UUID
	joinCode       JoinCode
	phase          GamePhase
	winningPlayers []Player
	suspension     *PlayerSuspension
	cards          []Card
	events         []Event
}
