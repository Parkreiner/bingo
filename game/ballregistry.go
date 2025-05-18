package game

import (
	"errors"
	"fmt"
	"sync"

	"github.com/Parkreiner/bingo"
)

// ballRegistry manages all bingo balls in a round of bingo. The registry can be
// reused across multiple rounds.
type ballRegistry struct {
	called   []bingo.Ball
	uncalled []bingo.Ball
	shuffler *shuffler
	mtx      *sync.Mutex
}

// newBallRegistry creates a new instance of a bingo ball registry
func newBallRegistry(rngSeed int64) *ballRegistry {
	shuffler := newShuffler(rngSeed)
	uncalled := generateBingoBallsForRange(1, 75)
	shuffler.shuffleBalls(uncalled)

	return &ballRegistry{
		called:   nil,
		uncalled: uncalled,
		shuffler: shuffler,
		mtx:      &sync.Mutex{},
	}
}

// NextAutomaticCall has the registry produce the next value for a game of
// bingo. Helpful if you don't have any in-person bingo ball machines.
func (br *ballRegistry) NextAutomaticCall() (bingo.Ball, error) {
	br.mtx.Lock()
	defer br.mtx.Unlock()

	if len(br.uncalled) == 0 {
		return bingo.FreeSpace, errors.New("registry has no more bingo balls")
	}

	l := len(br.uncalled) - 1
	next := br.uncalled[l]
	br.uncalled = br.uncalled[0:l]
	return next, nil
}

// SyncManualCall tells the registry which bingo ball was just called from an
// in-person bingo machine.
func (br *ballRegistry) SyncManualCall(ball bingo.Ball) error {
	br.mtx.Lock()
	defer br.mtx.Unlock()

	foundIndex := -1
	for i, b := range br.uncalled {
		if b == ball {
			foundIndex = i
			break
		}
	}
	if foundIndex == -1 {
		return fmt.Errorf("could not find bingo ball %d in list of uncalled bingo balls", ball)
	}

	br.called = append(br.called, br.uncalled[foundIndex])
	end := len(br.uncalled) - 1
	for i := foundIndex; i < end; i++ {
		br.uncalled[i] = br.uncalled[i+1]
	}
	br.uncalled = br.uncalled[0:end]

	return nil
}

// Reset reverts the state of the bingo ball registry to its initial state.
// Should be called at the start of each round of bingo.
func (br *ballRegistry) Reset() {
	br.mtx.Lock()
	defer br.mtx.Unlock()

	newUncalled := generateBingoBallsForRange(1, 75)
	br.shuffler.shuffleBalls(newUncalled)
	br.uncalled = newUncalled
	br.called = nil
}
