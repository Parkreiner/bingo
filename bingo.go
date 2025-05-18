// Package bingo contains the main domain types (and associated helper values
// and functions) needed to play a game of American bingo.
package bingo

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

const (
	// MinCards represents the minimum number of cards a player is allowed to
	// have in a game.
	MinCards int = 1

	// MaxCards represents the maximum number of cards a player is allowed to
	// have in a game.
	MaxCards int = 6

	// MaxBallValue is the highest bingo ball value that you can possibly have
	// in an American game of bingo
	MaxBallValue int = 75
)

// Ball represents a single number from 1 to 75 (both inclusive) that can
// called during a bingo game, as well as the free space (represented via the
// zero value)
type Ball byte

// FreeSpace represents the space given for free to all players. It is the zero
// value of Ball. It should not be daubed automatically on a bingo card, just so
// that players have more opportunities to stay engaged with the game UI
const FreeSpace = Ball(0)

var _ json.Marshaler = FreeSpace

// MarshalJSON turns a ball (which is already a byte) into a "human-readable"
// int, and then into a byte slice.
func (b Ball) MarshalJSON() ([]byte, error) {
	return json.Marshal(int(b))
}

// ParseBall takes any arbitrary int, and attempts to turn it into a bingo ball.
// Will error if the provided value is above 75 or below 0
func ParseBall(rawBallValue int) (Ball, error) {
	if rawBallValue > MaxBallValue {
		return FreeSpace, fmt.Errorf("value %d is not allowed to exceed %d", rawBallValue, MaxBallValue)
	}
	if rawBallValue < 0 {
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
	PlayerID uuid.UUID `json:"playerId"`
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
	// initialized. The game will be unable to be used, and it is assumed that
	// any attempts to subscribe to a game in this state should fail.
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

// PlayerStatus indicates the status of a player
type PlayerStatus string

const (
	// PlayerStatusHost indicates that a given player is hosting the game. That
	// player should have exclusive access to specific commands, and should
	// never be treated as a participant who has cards.
	PlayerStatusHost PlayerStatus = "host"
	// PlayerStatusActive indicates that a player is actively playing the game.
	PlayerStatusActive PlayerStatus = "active"
	// PlayerStatusWaitlisted indicates that a player has joined a game, but is
	// waiting for the next round to start.
	PlayerStatusWaitlisted PlayerStatus = "waitlisted"
	// PlayerStatusSuspended indicates that a player is part of a game, but has
	// acted out, and is waiting to have a suspension elapse before being able
	// to re-join the game.
	PlayerStatusSuspended PlayerStatus = "suspended"
	// PlayerStatusBanned indicates that a player has been banned. They should
	// not ever be allowed to re-join a game. This status likely won't be used
	// much in practice, but is included for completion's sake.
	PlayerStatusBanned PlayerStatus = "banned"
)

// Player represents any user who is able to join a game, either as a host or a
// card-player. If a player is host, their Cards field will be nil/empty.
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
	// Make sure the slice is allocated and can't be serialized as JSON null
	if copied.Cards == nil {
		copied.Cards = []*Card{}
	}
	return json.Marshal(copied)
}

// PlayerSuspension represents how long a player will be in time out for being
// a pain in the butt to the other users in the room
type PlayerSuspension struct {
	PlayerID      uuid.UUID `json:"playerId"`
	RoundDuration int       `json:"duration"`
	RoundsPassed  int       `json:"currentRound"`
}

// PhaseSubscriber is anything that lets a system listen to all events that can
// be dispatched for each possible bingo game phase.
type PhaseSubscriber interface {
	// Subscribe lets any external system subscribe to events generated during
	// specific game phases. If the provided slice is nil or empty, that causes
	// the system to subscribe to ALL events.
	Subscribe(phases []GamePhase) (eventReceiver <-chan GameEvent, unsubscribe func(), err error)
}

// GameManager is a stateful representation of a bingo game. It is able to
// receive direct user input, and also let external users subscribe to changes
// in the game state
type GameManager interface {
	PhaseSubscriber

	// IssueCommand allows an entity (e.g., a system, a host, or a player) to
	// make a command to update the game state. The GameManager implementation
	// should make sure that the command is valid for that entity to make, and
	// should handle all possible race conditions from multiple entities calling
	// IssueCommand at the same time.
	//
	// IssueCommand is intended as a lower-level primitive for processing all
	// the possible types of input that can be added to a game of bingo. It
	// should *not* be connected directly to user input.
	//
	// If the struct implementing the interface does not support a command, it
	// should return ErrCommandNotSupported.
	IssueCommand(cmd GameCommand) error
	// JoinGame allows a user to join a game and become a player. The resulting
	// player struct will have the same ID provided as input. Should error out
	// if a host tries to join a game they're currently hosting
	JoinGame(playerID uuid.UUID, playerName string) (player *Player, leaveGame func() error, err error)
}
