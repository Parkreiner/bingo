// Package bingo contains the main domain logic for playing a bingo game.
package bingo

import (
	"encoding/json"
	"fmt"
	"strconv"

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
// called during a bingo game, as well as the free space (represented via the
// zero value)
type Ball byte

// FreeSpace represents the space given for free to all players. It is the zero
// value of Ball. It should not be daubed automatically on a bingo card, just so
// that players have more to do in a round
const FreeSpace = Ball(0)

var _ json.Marshaler = FreeSpace

// MarshalJSON turns a ball (which is already a byte) into a "human-readable"
// byte. That is, the ball is converted to a string, and then to a byte slice.
func (b Ball) MarshalJSON() ([]byte, error) {
	text := strconv.Itoa(int(b))
	return []byte(text), nil
}

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
	Number Ball `json:"number"`
	// Indicates whether a player has marked the cell with a virtual dauber.
	// This value is allowed to be mutated.
	Daubed bool `json:"daubed"`
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
	Cells    [][]*Cell `json:"cells"`
	ID       uuid.UUID `json:"id"`
	PlayerID uuid.UUID `json:"player_id"`
}

const maxPlayerCapacity = 50

// GamePhase represents the current phase of a game. There can only be one phase
// at a time, and all phase constants are listed in the order that they proceed
// in while a game is ongoing.
type GamePhase string

const (
	// GamePhaseInitialized represents when a new Game instance has just been
	// created, but it hasn't been connected to any other parts of the program,
	// and is effectively inert. Once the game leaves this phase, it cannot ever
	// return to this phase
	GamePhaseInitialized GamePhase = "initialized"
	// GamePhaseInitializationFailure indicates that a game could not be
	// initialized. The game will be unable to be used.
	GamePhaseInitializationFailure GamePhase = "initialization_failure"
	// GamePhaseRoundStart represents when a new round has just started. It can
	// be considered an upkeep step for updating state that only updates at the
	// start of each round. This is the only phase when players are able to join
	// a game as participants. If a player joins in any other phase, they will
	// be waitlisted and will have to wait until the game enters
	// GamePhaseRoundStart again
	GamePhaseRoundStart GamePhase = "round_start"
	// GamePhaseCalling represents when a host is calling bingo balls for other
	// players to daub on their cards. It should generally be the
	// longest-running phase in the entire game
	GamePhaseCalling GamePhase = "calling"
	// GamePhaseConfirmingBingo represents when one or more players has called
	// bingo, and the host is validating that the user(s) are correct
	GamePhaseConfirmingBingo GamePhase = "confirming_bingo"
	// GamePhaseTiebreaker represents when more than one player has called
	// bingo, and we need a way to settle who actually wins the prize. It is up
	// to the host to decide whether only one or all the players wins the round,
	// and the game can be settled without making any more bingo calls
	GamePhaseTiebreaker GamePhase = "tiebreaker"
	// GamePhaseRoundEnd represents when a new round has just ended. It can
	// be considered an upkeep step for updating state that only updates at the
	// end of each round. It is assumed that this phase is entirely automatic,
	// and will transition to GamePhaseRoundStart without any player or host
	// intervention
	GamePhaseRoundEnd GamePhase = "round_end"
	// GamePhaseGameOver represents when the game has been terminated, either
	// via manual termination, or by having the game end naturally. Once the
	// game enters this phase, it cannot transition to any other phases. A new
	// host will need to start a new game from scratch.
	GamePhaseGameOver GamePhase = "game_over"
)

// AllGamePhases is a slice of all game phases. It should be treated as a
// readonly slice for the entire lifetime of the program.
var AllGamePhases = []GamePhase{
	GamePhaseInitialized,
	GamePhaseInitializationFailure,
	GamePhaseRoundStart,
	GamePhaseCalling,
	GamePhaseConfirmingBingo,
	GamePhaseTiebreaker,
	GamePhaseRoundEnd,
	GamePhaseGameOver,
}

// PlayerStatus indicates the status of a player
type PlayerStatus string

const (
	PlayerStatusHost       PlayerStatus = "host"
	PlayerStatusActive     PlayerStatus = "active"
	PlayerStatusWaitlisted PlayerStatus = "waitlisted"
	PlayerStatusSuspended  PlayerStatus = "suspended"
	PlayerStatusBanned     PlayerStatus = "banned"
)

// Player represents any user who is able to join a game as a card-player.
type Player struct {
	Status        PlayerStatus
	ID            uuid.UUID
	Name          string
	Cards         []*Card
	EventReceiver <-chan GameEvent
}

var _ json.Marshaler = &Player{}

// MarshalJSON serializes all values from Player that are safe to serialize, and
// skips over the rest
func (p *Player) MarshalJSON() ([]byte, error) {
	type PlayerWithoutReceiver struct {
		ID     uuid.UUID    `json:"id"`
		Name   string       `json:"name"`
		Cards  []*Card      `json:"cards"`
		Status PlayerStatus `json:"status"`
	}
	copied := PlayerWithoutReceiver{
		ID:    p.ID,
		Name:  p.Name,
		Cards: p.Cards,
	}
	if copied.Cards == nil {
		copied.Cards = []*Card{}
	}
	return json.Marshal(copied)
}

// PlayerSuspension represents how long a player will be in time out for being
// a pain in the butt to the other users in the room
type PlayerSuspension struct {
	PlayerID      uuid.UUID `json:"player_id"`
	RoundDuration int       `json:"duration"`
	RoundsPassed  int       `json:"current_round"`
}

// GameManager is a stateful representation of a bingo game. It is able to
// receive direct user input, and also let external users subscribe to changes
// in the game state
type GameManager interface {
	// Command allows an entity (e.g., a system, a host, or a player) to make a
	// command to update the game state. The GameManager implementation should
	// make sure that the command is valid for that entity to make, and should
	// handle all possible race conditions from multiple entities calling
	// Command at the same time.
	//
	// Command is intended as a lower-level primitive for processing all the
	// possible events that can happen in a game of bingo. It should not be
	// connected directly to user input.
	Command(cmd GameCommand) error
	// JoinGame allows a user to join a game and become a player. The resulting
	// player struct will have the same ID provided as input. Should error out
	// if a host tries to join a game they're currently hosting
	JoinGame(playerID uuid.UUID, playerName string) (player *Player, leaveGame func() error, err error)
	// SubscribeToPhaseEvents allows an external system to subscribe to all
	// events for a given phase type. An easy way to subscribe to all events is
	// to iterate over the AllGamePhases slice, and call this method for each
	// element
	SubscribeToPhaseEvents(phase GamePhase) (eventReceiver <-chan GameEvent, unsubscribe func(), err error)
}
