// Package game defines the minimal implementation for a full bingo game
package game

import (
	"errors"
	"fmt"

	"github.com/Parkreiner/bingo"
	"github.com/google/uuid"
)

// Game is a minimal implementation of the bingo.Game interface
type Game struct {
	id                uuid.UUID
	currentRound      int
	maxRounds         int
	phase             bingo.GamePhase
	ballRegistry      BallRegistry
	cardRegistry      CardRegistry
	host              bingo.User
	activePlayers     []bingo.User
	waitlistedPlayers []bingo.User

	// This value keeps track of which player was responsible for winning a
	// given round. The whole player is stored because it's possible for a
	// player to leave the game, so there's no guarantee that an ID in
	// winningPlayers would match with the players field. This field cannot be
	// used to derive the round count, because it's possible for multiple
	// players to win in a single round.
	winningPlayers       []bingo.User
	bingoCallerPlayerIDs []uuid.UUID
	suspensions          []bingo.PlayerSuspension
	bannedPlayerIDs      []uuid.UUID
	terminationCallback  func()
}

var _ bingo.Game = &Game{}

func NewGame(host bingo.User, maxRounds int, rngSeed int64) *Game {
	return &Game{
		maxRounds:            maxRounds,
		currentRound:         0,
		host:                 host,
		id:                   uuid.New(),
		phase:                bingo.GamePhaseInitialized,
		ballRegistry:         *newBallRegistry(rngSeed),
		cardRegistry:         *newCardRegistry(rngSeed),
		activePlayers:        nil,
		waitlistedPlayers:    nil,
		winningPlayers:       nil,
		bingoCallerPlayerIDs: nil,
		suspensions:          nil,
		bannedPlayerIDs:      nil,
		terminationCallback:  nil,
	}
}

func (g *Game) Start() error {
	if g.phase != bingo.GamePhaseInitialized {
		return errors.New("game has already been started")
	}

	cleanupCardRegistry, err := g.cardRegistry.Start()
	if err != nil {
		return fmt.Errorf("unable to start game: %v", err)
	}

	g.terminationCallback = func() {
		cleanupCardRegistry()
	}
	return nil
}

func (g *Game) Terminate() error {
	if g.phase == bingo.GamePhaseInitialized {
		return errors.New("game must be started before it can be terminated")
	}
	if g.terminationCallback == nil {
		return errors.New("unable to terminate game")
	}

	g.terminationCallback()
	g.phase = bingo.GamePhaseGameOver
	return nil
}
