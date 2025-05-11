// Package game defines the minimal implementation for a full, stateful,
// multiplayer bingo game
package game

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sync"

	"github.com/Parkreiner/bingo"
	"github.com/google/uuid"
)

const (
	defaultMaxRounds  = 8
	defaultMaxPlayers = 50
)

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
	activePlayers        []bingo.Player
	waitlistedPlayers    []bingo.Player
	id                   uuid.UUID
	phase                bingo.GamePhase
	creatorID            uuid.UUID
	currentRound         int
	maxRounds            int
	maxPlayers           int
	dispose              func()
	commandChan          chan bingo.GameCommand
	mtx                  sync.Mutex
	// It is assumed that the map will be initialized with one entry per game
	// phase when a new game is instantiated
	phaseSubscriptions map[bingo.GamePhase][]chan bingo.GameEvent
}

var _ bingo.GameManager = &Game{}

type GameInit struct {
	creatorID  uuid.UUID
	host       bingo.User
	rngSeed    int64
	maxPlayers *int
	maxRounds  *int
}

// New creates a new instance of a Game
func New(init GameInit) (*Game, error) {
	game := &Game{
		creatorID:    init.creatorID,
		host:         init.host,
		maxRounds:    defaultMaxRounds,
		maxPlayers:   defaultMaxPlayers,
		ballRegistry: *newBallRegistry(init.rngSeed),
		cardRegistry: *newCardRegistry(init.rngSeed),

		// Unbuffered to have synchronization guarantees
		commandChan:          make(chan bingo.GameCommand),
		phaseSubscriptions:   make(map[bingo.GamePhase][]chan bingo.GameEvent),
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
	if init.maxPlayers != nil {
		game.maxPlayers = *init.maxPlayers
	}

	// Make sure to do things that can fail first, before we get too far into
	// the initialization
	terminateCardRegistry, err := game.cardRegistry.Start()
	if err != nil {
		return nil, fmt.Errorf("initializing game: %v", err)
	}

	for _, gp := range bingo.AllGamePhases {
		game.phaseSubscriptions[gp] = nil
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
	loop:
		for {
			select {
			case event, closed := <-game.commandChan:
				if closed {
					break loop
				}
				game.processQueuedCommand(event)
			}
		}
	}()

	return game, nil
}

func (g *Game) processQueuedCommand(bingo.GameCommand) error {
	return nil
}

func (g *Game) JoinGame(id uuid.UUID) (*bingo.Player, func(), error) {
	g.mtx.Lock()
	defer g.mtx.Unlock()

	if id == g.host.ID {
		return nil, nil, fmt.Errorf("player cannot join game that they are hosting")
	}
	if id == g.creatorID {
		return nil, nil, fmt.Errorf("trying to add ID that belongs to system. Something is very wrong")
	}

	alreadyAdded := slices.ContainsFunc(g.activePlayers, func(p bingo.Player) bool {
		return p.User.ID == id
	})
	if alreadyAdded {
		// Todo: Figure out how to make sure that calling an unsubscribe
		// callback multiple times won't break things
	}

	return nil, func() {}, nil
}

func (g *Game) SubscribeToPhaseEvents(bingo.GamePhase) (<-chan bingo.GameEvent, func(), error) {
	return nil, func() {}, nil
}

func (g *Game) SubscribeToAllPhaseEvents() (<-chan bingo.GameEvent, func(), error) {
	g.mtx.Lock()
	defer g.mtx.Unlock()

	if g.phase == bingo.GamePhaseGameOver {
		return nil, nil, errors.New("cannot subscribe to game that has been terminated")
	}

	var phaseEmitters []chan bingo.GameEvent
	for _, gp := range bingo.AllGamePhases {
		newEmitter := make(chan bingo.GameEvent)
		phaseEmitters = append(phaseEmitters, newEmitter)
		g.phaseSubscriptions[gp] = append(g.phaseSubscriptions[gp], newEmitter)
	}

	consolidatedEmitter := make(chan bingo.GameEvent)
	go func() {
		var selectCases []reflect.SelectCase
		for _, pe := range phaseEmitters {
			selectCases = append(selectCases, reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(pe),
			})
		}

		closeCount := 0
		for {
			_, value, closed := reflect.Select(selectCases)
			if closed {
				closeCount++
				if closeCount == len(phaseEmitters)-1 {
					break
				}
				continue
			}

			converted, ok := value.Interface().(bingo.GameEvent)
			if !ok {
				break
			}
			consolidatedEmitter <- converted
		}
	}()

	cleanup := func() {
		g.mtx.Lock()
		defer g.mtx.Unlock()

		for i, phase := range bingo.AllGamePhases {
			emitterToDispose := phaseEmitters[i]
			var filtered []chan bingo.GameEvent
			for _, emitter := range g.phaseSubscriptions[phase] {
				if emitter != emitterToDispose {
					filtered = append(filtered, emitter)
				}
			}
			g.phaseSubscriptions[phase] = filtered
			close(emitterToDispose)
		}

		close(consolidatedEmitter)
	}

	return consolidatedEmitter, cleanup, nil
}

func (g *Game) Command(bingo.GameCommand) error {
	return nil
}
