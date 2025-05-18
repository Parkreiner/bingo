// Package game defines the minimal implementation for a full, stateful,
// multiplayer bingo game
package game

import (
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/Parkreiner/bingo"
	"github.com/google/uuid"
)

var errTodo = errors.New("not implemented yet")

const (
	defaultMaxRounds  = 8
	defaultMaxPlayers = 50
)

// playerEntry contains information about a player that is currently part of the
// game, as well as any state necessary for that player to perform actions
// (including leaving the game)
type playerEntry struct {
	leaveGame func() error
	player    *bingo.Player
}

type commandSession struct {
	command   bingo.GameCommand
	errorChan chan<- error
}

// Game is an implementation of the bingo.GameManager interface
// TODO: Figure out how to split this struct up so that there's less contention
// for the mutex locks. That, or figure out a way to do EVERYTHING with channels
type Game struct {
	cardRegistry cardRegistry
	ballRegistry ballRegistry
	host         *bingo.Player
	// cardPlayers represents all the players currently in the game (minus the
	// host)
	cardPlayers []*playerEntry
	// bingoCallerPlayerIDs refers to all players who are currently claiming to
	// have bingo.
	bingoCallerPlayerIDs []uuid.UUID
	// winningPlayers keeps track of which player(s) were responsible for
	// winning a given round. The whole player is stored because it's possible
	// for a player to leave the game, so there's no guarantee that an ID in
	// winningPlayers would match with the cardPlayers field. This field cannot
	// be used to derive the round count, because it's possible for multiple
	// players to win in a single round.
	winningPlayers     []*bingo.Player
	suspensions        []*bingo.PlayerSuspension
	bannedPlayerIDs    []uuid.UUID
	phase              phase
	systemID           uuid.UUID
	currentRound       int
	maxRounds          int
	maxPlayers         int
	dispose            func() error
	commandChan        chan commandSession
	mtx                sync.Mutex
	phaseSubscriptions subscriptionsManager
}

var _ bingo.GameManager = &Game{}

// Init is used to instantiate a Game instance via the New function
type Init struct {
	creatorID  uuid.UUID
	hostID     uuid.UUID
	hostName   string
	rngSeed    int64
	maxPlayers *int
	maxRounds  *int
}

// New creates a new instance of a Game
func New(init Init) (*Game, error) {
	host := &bingo.Player{
		Status:        bingo.PlayerStatusHost,
		ID:            init.hostID,
		Name:          init.hostName,
		Cards:         nil,
		EventReceiver: nil,
	}

	game := &Game{
		systemID:           init.creatorID,
		host:               host,
		maxRounds:          defaultMaxRounds,
		maxPlayers:         defaultMaxPlayers,
		ballRegistry:       *newBallRegistry(init.rngSeed),
		cardRegistry:       *newCardRegistry(init.rngSeed),
		phaseSubscriptions: newSubscriptionsManager(),

		// Unbuffered to have synchronization guarantees
		commandChan:          make(chan commandSession),
		phase:                newPhase(),
		currentRound:         0,
		cardPlayers:          nil,
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
		game.phase.setValue(bingo.GamePhaseInitializationFailure)
		return nil, fmt.Errorf("failed to initialize: %v", err)
	}

	disposed := false
	game.dispose = func() error {
		if disposed {
			return nil
		}

		close(game.commandChan)
		terminateCardRegistry()
		err := game.phaseSubscriptions.dispose(game.systemID)
		disposed = true
		return err
	}

	go func() {
		for session := range game.commandChan {
			err := game.routeCommand(session.command)
			session.errorChan <- err
		}
	}()

	return game, nil
}

func (g *Game) routeCommand(command bingo.GameCommand) error {
	switch command.Type {
	// System commands
	case bingo.GameCommandSystemDispose:
		return g.processSystemDispose(command.CommanderID)
	case bingo.GameCommandSystemBroadcastState:
		return g.processSystemBroadcastState(command.CommanderID)

	// Host commands
	case bingo.GameCommandHostStartGame:
		return errTodo
	case bingo.GameCommandHostTerminateGame:
		return errTodo
	case bingo.GameCommandHostBanPlayer:
		return errTodo
	case bingo.GameCommandHostSuspendPlayer:
		return errTodo
	case bingo.GameCommandHostRequestBall:
		return errTodo
	case bingo.GameCommandHostSyncBall:
		return errTodo
	case bingo.GameCommandHostAcknowledgeBingoCall:
		return errTodo
	case bingo.GameCommandHostStartTiebreakerRound:
		return errTodo
	case bingo.GameCommandHostAwardPlayers:
		return errTodo

	// Player commands
	case bingo.GameCommandPlayerDaub:
		return g.processPlayerDaub(command)
	case bingo.GameCommandPlayerUndoDaub:
		return g.processPlayerUndoDaub(command)
	case bingo.GameCommandPlayerCallBingo:
		return errTodo
	case bingo.GameCommandPlayerRescindBingo:
		return errTodo
	case bingo.GameCommandPlayerReplaceCards:
		return g.processHandReplacement(command.CommanderID)

	default:
		return fmt.Errorf("received unknown command %q", command.Type)
	}
}

// JoinGame allows a player to join a game as a normal player. The method will
// prevent a player with the same ID from joining a game multiple times. If the
// join attempt is successful, the returned player will be given a full hand of
// bingo cards, ready to use.
//
// The returned callback lets a user leave the game. Calling the callback more
// than once results in a no-op.
func (g *Game) JoinGame(playerID uuid.UUID, playerName string) (*bingo.Player, func() error, error) {
	g.mtx.Lock()
	defer g.mtx.Unlock()

	if !g.phase.ok() {
		return nil, nil, errors.New("cannot join game that has been terminated")
	}
	if playerID == g.host.ID {
		return nil, nil, errors.New("player cannot join game that they are hosting")
	}
	if playerID == g.systemID {
		return nil, nil, errors.New("trying to add ID that belongs to system. Something is very wrong")
	}
	if slices.Contains(g.bannedPlayerIDs, playerID) {
		return nil, nil, fmt.Errorf("player ID %q is banned", playerID)
	}

	// Only make a new entry if it doesn't exist in the game at all
	var prevEntry *playerEntry
	for _, e := range g.cardPlayers {
		if e.player.ID == playerID {
			prevEntry = e
			break
		}
	}
	if prevEntry != nil {
		return prevEntry.player, prevEntry.leaveGame, nil
	}

	eventChan, unsub, err := g.phaseSubscriptions.subscribe(nil, []uuid.UUID{playerID})
	if err != nil {
		return nil, nil, fmt.Errorf("unable to join game: %v", err)
	}

	var cards []*bingo.Card
	for i := 0; i < bingo.MaxCards; i++ {
		card, err := g.cardRegistry.CheckOutCard(playerID)
		if err != nil {
			unsub()
			return nil, nil, fmt.Errorf("unable to produce card %d for player %q (ID %s): %v", i+1, playerName, playerID, err)
		}
		cards = append(cards, card)
	}
	status := bingo.PlayerStatusWaitlisted
	if g.phase.value() == bingo.GamePhaseRoundStart {
		status = bingo.PlayerStatusActive
	}
	player := &bingo.Player{
		Status:        status,
		ID:            playerID,
		Name:          playerName,
		Cards:         cards,
		EventReceiver: eventChan,
	}

	leftGame := false
	newEntry := &playerEntry{
		player: player,
		leaveGame: func() error {
			if leftGame {
				return nil
			}

			g.mtx.Lock()
			defer g.mtx.Unlock()

			var removedEntry *playerEntry
			var remainder []*playerEntry
			for _, e := range g.cardPlayers {
				if e.player.ID == playerID {
					removedEntry = e
				} else {
					remainder = append(remainder, e)
				}
			}
			if len(remainder) == len(g.cardPlayers) {
				return nil
			}

			g.cardPlayers = remainder
			var cardReturnErr error
			for _, card := range removedEntry.player.Cards {
				// Don't stop at the first error found, because there's a chance
				// that the other cards can still be returned/recycled for
				// future rounds with other players
				err := g.cardRegistry.ReturnCard(card.ID)
				if err != nil {
					cardReturnErr = err
				}
			}

			unsub()
			leftGame = true
			return cardReturnErr
		},
	}

	g.cardPlayers = append(g.cardPlayers, newEntry)
	return newEntry.player, newEntry.leaveGame, nil
}

// Subscribe lets an external system subscribe to all events emitted during
// specific game phases. If the provided slice is nil or empty, that causes the
// system to subscribe to ALL events for ALL game phases.
func (g *Game) Subscribe(phases []bingo.GamePhase) (<-chan bingo.GameEvent, func(), error) {
	if !g.phase.ok() {
		return nil, nil, errors.New("game is not able to accept new subscriptions")
	}

	return g.phaseSubscriptions.subscribe(phases, nil)
}

// IssueCommand allows the Game to receive direct input from outside sources
func (g *Game) IssueCommand(command bingo.GameCommand) error {
	if !g.phase.ok() {
		return errors.New("game is not able to accept new commands")
	}

	channel := make(chan error)
	defer close(channel)
	g.commandChan <- commandSession{
		command:   command,
		errorChan: channel,
	}

	err := <-channel
	return err
}
