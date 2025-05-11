// Package game defines the minimal implementation for a full, stateful,
// multiplayer bingo game
package game

import (
	"fmt"

	"github.com/Parkreiner/bingo"
	"github.com/google/uuid"
)

const defaultMaxRounds = 8

// Game is an implementation of the bingo.GameManager interface
type Game struct {
	cardRegistry CardRegistry
	ballRegistry BallRegistry
	host         bingo.User
	// winningPlayers keeps track of which player(s) were responsible for
	// winning a given round. The whole player is stored because it's possible
	// for a player to leave the game, so there's no guarantee that an ID in
	// winningPlayers would match with the players field. This field cannot be
	// used to derive the round count, because it's possible for multiple
	// players to win in a single round.
	winningPlayers       []*bingo.Player
	bingoCallerPlayerIDs []uuid.UUID
	suspensions          []bingo.PlayerSuspension
	bannedPlayerIDs      []uuid.UUID
	activePlayers        []bingo.User
	waitlistedPlayers    []bingo.User
	id                   uuid.UUID
	phase                bingo.GamePhase
	creatorID            uuid.UUID
	currentRound         int
	maxRounds            int
	dispose              func()
	commandChan          chan bingo.GameCommand
}

var _ bingo.GameManager = &Game{}

type GameInit struct {
	creatorID uuid.UUID
	host      bingo.User
	rngSeed   int64
	maxRounds *int
}

// New creates a new instance of a Game
func New(init GameInit) (*Game, error) {
	game := &Game{
		creatorID:    init.creatorID,
		host:         init.host,
		maxRounds:    defaultMaxRounds,
		ballRegistry: *newBallRegistry(init.rngSeed),
		cardRegistry: *newCardRegistry(init.rngSeed),

		// Unbuffered to have synchronization guarantees
		commandChan:          make(chan bingo.GameCommand),
		phase:                bingo.GamePhaseInitialized,
		id:                   uuid.New(),
		currentRound:         0,
		activePlayers:        nil,
		waitlistedPlayers:    nil,
		winningPlayers:       nil,
		bingoCallerPlayerIDs: nil,
		suspensions:          nil,
		bannedPlayerIDs:      nil,
		dispose:              nil,
	}

	if init.maxRounds != nil {
		game.maxRounds = *init.maxRounds
	}

	terminateCardRegistry, err := game.cardRegistry.Start()
	if err != nil {
		return nil, fmt.Errorf("initializing game: %v", err)
	}

	disposed := false
	game.dispose = func() {
		if disposed {
			return
		}
		close(game.commandChan)
		terminateCardRegistry()
		disposed = true
	}

	go func() {
		for {
			select {
			case event := <-game.commandChan:
				game.processQueuedCommand(event)
			}
		}
	}()

	return game, nil
}

func (g *Game) processQueuedCommand(bingo.GameCommand) error {
	return nil
}

func (g *Game) SubscribeToEntityEvents(uuid.UUID) (<-chan bingo.GameEvent, func(), error) {
	return nil, func() {}, nil
}

func (g *Game) SubscribeToPhaseEvents(bingo.GamePhase) (<-chan bingo.GameEvent, func(), error) {
	return nil, func() {}, nil
}

func (g *Game) Command(bingo.GameCommand) error {
	return nil
}
