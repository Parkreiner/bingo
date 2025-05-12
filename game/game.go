// Package game defines the minimal implementation for a full, stateful,
// multiplayer bingo game
package game

import (
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/Parkreiner/bingo"
	"github.com/google/uuid"
)

const (
	defaultMaxRounds  = 8
	defaultMaxPlayers = 50
)

// playerEntry contains information about a player that is currently part of the
// game, as well as any state necessary for that player to perform actions
// (including leaving the game)
type playerEntry struct {
	// eventChan should always be the same event channel attached to the
	// player field. The main difference is that this version allows two-way
	// channel communication, while the player version enforces receive-only
	// communication at the type level
	eventChan chan bingo.GameEvent
	leaveGame func() error
	player    *bingo.Player
}

type commandSession struct {
	command   bingo.GameCommand
	errorChan chan<- error
}

// Game is an implementation of the bingo.GameManager interface
type Game struct {
	cardRegistry cardRegistry
	ballRegistry ballRegistry
	host         *bingo.Player
	// cardPlayerEntries represents all the players currently in the game (minus the
	// host)
	cardPlayerEntries []*playerEntry
	// bingoCallerPlayerIDs refers to all players who are currently claiming to
	// have bingo.
	bingoCallerPlayerIDs []uuid.UUID
	// winningPlayers keeps track of which player(s) were responsible for
	// winning a given round. The whole player is stored because it's possible
	// for a player to leave the game, so there's no guarantee that an ID in
	// winningPlayers would match with the cardPlayers field. This field cannot
	// be used to derive the round count, because it's possible for multiple
	// players to win in a single round.
	winningPlayers  []*bingo.Player
	suspensions     []*bingo.PlayerSuspension
	bannedPlayerIDs []uuid.UUID
	id              uuid.UUID
	phase           bingo.GamePhase
	systemID        uuid.UUID
	currentRound    int
	maxRounds       int
	maxPlayers      int
	dispose         func()
	commandChan     chan commandSession
	mtx             sync.Mutex
	// It is assumed that the map will be initialized with one entry per game
	// phase when a new game is instantiated
	phaseSubscriptions map[bingo.GamePhase][]chan bingo.GameEvent
}

var _ bingo.GameManager = &Game{}

// Init is used to instantiate a Game instance via the New function
type Init struct {
	systemID   uuid.UUID
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
		systemID:     init.systemID,
		host:         host,
		maxRounds:    defaultMaxRounds,
		maxPlayers:   defaultMaxPlayers,
		ballRegistry: *newBallRegistry(init.rngSeed),
		cardRegistry: *newCardRegistry(init.rngSeed),

		// Unbuffered to have synchronization guarantees
		commandChan:          make(chan commandSession),
		phaseSubscriptions:   make(map[bingo.GamePhase][]chan bingo.GameEvent),
		id:                   uuid.New(),
		phase:                bingo.GamePhaseInitialized,
		currentRound:         0,
		cardPlayerEntries:    nil,
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
		game.phase = bingo.GamePhaseInitializationFailure
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
		for session := range game.commandChan {
			err := game.processQueuedCommand(session.command)
			session.errorChan <- err
		}
	}()

	return game, nil
}

func (g *Game) processQueuedCommand(bingo.GameCommand) error {
	return nil
}

// JoinGame allows a player to join a game as a normal player. Trying to
func (g *Game) JoinGame(playerID uuid.UUID, playerName string) (*bingo.Player, func() error, error) {
	g.mtx.Lock()
	defer g.mtx.Unlock()

	if g.phase == bingo.GamePhaseGameOver {
		return nil, nil, fmt.Errorf("cannot join game that has been terminated")
	}
	if playerID == g.host.ID {
		return nil, nil, fmt.Errorf("player cannot join game that they are hosting")
	}
	if playerID == g.systemID {
		return nil, nil, fmt.Errorf("trying to add ID that belongs to system. Something is very wrong")
	}

	// Only make a new entry if it doesn't exist in the game at all
	var prevEntry *playerEntry
	for _, e := range g.cardPlayerEntries {
		if e.player.ID == playerID {
			prevEntry = e
			break
		}
	}
	if prevEntry != nil {
		return prevEntry.player, prevEntry.leaveGame, nil
	}

	var cards []*bingo.Card
	for i := 0; i < bingo.MaxCards; i++ {
		card, err := g.cardRegistry.CheckOutCard(playerID)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to produce card %d for %q (%s): %v", i+1, playerID, playerName, err)
		}
		cards = append(cards, card)
	}

	// Very important that the same eventChan be embedded in both the entry and
	// the player inside the entry
	eventChan := make(chan bingo.GameEvent)
	status := bingo.PlayerStatusWaitlisted
	if g.phase == bingo.GamePhaseRoundStart {
		status = bingo.PlayerStatusActive
	}
	newEntry := &playerEntry{
		eventChan: eventChan,
		player: &bingo.Player{
			Status:        status,
			ID:            playerID,
			Name:          playerName,
			Cards:         cards,
			EventReceiver: eventChan,
		},
		leaveGame: nil,
	}
	newEntry.leaveGame = func() error {
		g.mtx.Lock()
		defer g.mtx.Unlock()

		var removedEntry *playerEntry
		var filtered []*playerEntry
		for _, e := range g.cardPlayerEntries {
			if e.player.ID == playerID {
				removedEntry = e
			} else {
				filtered = append(filtered, e)
			}
		}
		if len(filtered) == len(g.cardPlayerEntries) {
			return nil
		}

		g.cardPlayerEntries = filtered
		var cardReturnErr error
		for _, card := range removedEntry.player.Cards {
			// Don't stop at the first error found, because there's a chance
			// that the other cards can still be returned/recycled for future
			// rounds with other players
			err := g.cardRegistry.ReturnCard(card.ID)
			if err != nil {
				cardReturnErr = err
			}
		}

		close(newEntry.eventChan)
		return cardReturnErr
	}

	g.cardPlayerEntries = append(g.cardPlayerEntries, newEntry)
	return newEntry.player, newEntry.leaveGame, nil
}

// SubscribeToPhaseEvents lets an external system subscribe to all events
// emitted during a given phase. There is no filtering beyond that – if the game
// is in the phase that was subscribed to, ALL events for all users will be
// emitted
func (g *Game) SubscribeToPhaseEvents(phase bingo.GamePhase) (<-chan bingo.GameEvent, func(), error) {
	g.mtx.Lock()
	defer g.mtx.Unlock()

	if g.phase == bingo.GamePhaseGameOver {
		return nil, nil, errors.New("cannot subscribe to game that has been terminated")
	}

	newEmitter := make(chan bingo.GameEvent)
	g.phaseSubscriptions[phase] = append(g.phaseSubscriptions[phase], newEmitter)

	subscribed := true
	unsubscribe := func() {
		if !subscribed {
			return
		}

		g.mtx.Lock()
		defer g.mtx.Unlock()

		var filtered []chan bingo.GameEvent
		for _, emitter := range g.phaseSubscriptions[phase] {
			if emitter != newEmitter {
				filtered = append(filtered, emitter)
			}
		}

		g.phaseSubscriptions[phase] = filtered
		close(newEmitter)
		subscribed = false
	}

	return newEmitter, unsubscribe, nil
}

// SubscribeToAllEvents is a convenience method for subscribing to all
// possible phase events. It is fully equivalent to calling the
// SubscribeToPhaseEvents method once for each phase type, and then stitching
// the resulting return types together manually.
func (g *Game) SubscribeToAllEvents() (<-chan bingo.GameEvent, func(), error) {
	g.mtx.Lock()
	defer g.mtx.Unlock()

	var phaseEmitters []<-chan bingo.GameEvent
	var unsubCallbacks []func()
	for _, gp := range bingo.AllGamePhases {
		newEmitter, unsub, err := g.SubscribeToPhaseEvents(gp)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to subscribe for phase %q: %v", gp, err)
		}
		phaseEmitters = append(phaseEmitters, newEmitter)
		unsubCallbacks = append(unsubCallbacks, unsub)
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

	subscribed := true
	unsubscribe := func() {
		if !subscribed {
			return
		}

		g.mtx.Lock()
		defer g.mtx.Unlock()

		for _, cb := range unsubCallbacks {
			cb()
		}
		close(consolidatedEmitter)
		subscribed = false
	}

	return consolidatedEmitter, unsubscribe, nil
}

// Command allows the Game to receive direct input from outside sources
func (g *Game) Command(command bingo.GameCommand) error {
	g.mtx.Lock()
	defer g.mtx.Unlock()

	if g.phase == bingo.GamePhaseGameOver ||
		g.phase == bingo.GamePhaseInitializationFailure {
		return errors.New("cannot send command while game is terminated")
	}

	channel := make(chan error)
	defer close(channel)
	g.commandChan <- commandSession{
		command:   command,
		errorChan: channel,
	}

	err := <-channel
	if err == nil {
		return nil
	}

	return nil
}
