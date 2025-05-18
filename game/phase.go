package game

import (
	"errors"
	"sync"

	"github.com/Parkreiner/bingo"
)

type phase struct {
	_p bingo.GamePhase
	// It's expected that the phase will be read from FAR more often than it
	// will be written to, so a regular mutex doesn't make sense
	rwmtx *sync.RWMutex
}

func newPhase() phase {
	return phase{
		_p:    bingo.GamePhaseInitialized,
		rwmtx: &sync.RWMutex{},
	}
}

func (p *phase) getValue() bingo.GamePhase {
	p.rwmtx.RLock()
	defer p.rwmtx.RUnlock()
	return p._p
}

// ok provides a convenience check for whether a game is "generically okay". As
// in, it's generally able to accept new subscriptions and commands, but it's
// possible for there to be other errors within the game state.
func (p *phase) ok() bool {
	value := p.getValue()
	return value != bingo.GamePhaseGameOver && value != bingo.GamePhaseInitializationFailure
}

func (p *phase) setValue(newValue bingo.GamePhase) error {
	p.rwmtx.Lock()
	defer p.rwmtx.Unlock()

	switch p._p {
	case bingo.GamePhaseGameOver:
		return errors.New("game is over")
	case bingo.GamePhaseInitializationFailure:
		return errors.New("game failed to initialize")
	default:
		p._p = newValue
		return nil
	}
}
