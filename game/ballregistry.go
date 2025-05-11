package game

import (
	"errors"
	"fmt"
	"sync"

	"github.com/Parkreiner/bingo"
)

// BallRegistry manages all bingo balls in a round of bingo. The registry can be
// reused across multiple rounds.
type BallRegistry struct {
	called   []bingo.Ball
	uncalled []bingo.Ball
	shuffler *Shuffler
	mtx      *sync.Mutex
}

var _ bingo.BallRegistry = &BallRegistry{}

// NewRegistry creates a new instance of a bingo ball registry
func NewRegistry(rngSeed int64) *BallRegistry {
	shuffler := NewShuffler(rngSeed)
	uncalled := bingo.GenerateBingoBallsForRange(1, 75)
	shuffler.ShuffleBingoBalls(uncalled)

	return &BallRegistry{
		called:   nil,
		uncalled: uncalled,
		shuffler: shuffler,
		mtx:      &sync.Mutex{},
	}
}

// NextAutomaticCall has the registry produce the next value for a game of
// bingo. Helpful if you don't have any in-person bingo ball machines.
func (a *BallRegistry) NextAutomaticCall() (bingo.Ball, error) {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	if len(a.uncalled) == 0 {
		return bingo.FreeSpace, errors.New("registry has no more bingo balls")
	}

	l := len(a.uncalled) - 1
	next := a.uncalled[l]
	a.uncalled = a.uncalled[0:l]
	return next, nil
}

// SyncManualCall tells the registry which bingo ball was just called from an
// in-person bingo machine.
func (a *BallRegistry) SyncManualCall(ball bingo.Ball) error {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	foundIndex := -1
	for i, b := range a.uncalled {
		if b == ball {
			foundIndex = i
			break
		}
	}

	if foundIndex == -1 {
		return fmt.Errorf("could not find bingo ball %d in list of uncalled bingo balls", ball)
	}

	a.called = append(a.called, a.uncalled[foundIndex])
	end := len(a.uncalled) - 1
	for i := foundIndex; i < end; i++ {
		a.uncalled[i] = a.uncalled[i+1]
	}
	a.uncalled = a.uncalled[0:end]

	return nil
}

// Reset reverts the state of the bingo ball registry to its initial state.
// Should be called at the start of each round of bingo.
func (a *BallRegistry) Reset() {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	newUncalled := bingo.GenerateBingoBallsForRange(1, 75)
	a.shuffler.ShuffleBingoBalls(newUncalled)
	a.called = nil
	a.uncalled = newUncalled
}
