package game

import (
	"errors"
	"sync"

	"github.com/Parkreiner/bingo"
)

type phase struct {
	_value bingo.GamePhase
	// It's expected that the phase will be read from FAR more often than it
	// will be written to, so a regular mutex doesn't make sense
	rwmtx *sync.RWMutex
}

func newPhase() phase {
	return phase{
		_value: bingo.GamePhaseInitialized,
		rwmtx:  &sync.RWMutex{},
	}
}

func (p *phase) value() bingo.GamePhase {
	p.rwmtx.RLock()
	defer p.rwmtx.RUnlock()
	return p._value
}

// ok provides a convenience check for whether a game is "generically okay". As
// in, it's generally able to accept new subscriptions and commands, but the
// system cannot guarantee whether those subscriptions/commands will succeed.
func (p *phase) ok() bool {
	v := p.value()
	return v != bingo.GamePhaseGameOver && v != bingo.GamePhaseInitializationFailure
}

func (p *phase) setValue(newValue bingo.GamePhase) error {
	p.rwmtx.Lock()
	defer p.rwmtx.Unlock()

	switch p._value {
	case bingo.GamePhaseGameOver:
		return errors.New("game is over")
	case bingo.GamePhaseInitializationFailure:
		return errors.New("game failed to initialize")
	default:
		p._value = newValue
		return nil
	}
}
