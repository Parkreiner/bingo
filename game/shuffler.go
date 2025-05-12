package game

import (
	"math/rand"

	"github.com/Parkreiner/bingo"
)

// shuffler provides methods for shuffling bingo calls using seed-based random
// logic.
type shuffler struct {
	rng *rand.Rand
}

// newShuffler creates a new instance of a Shuffler
func newShuffler(rngSeed int64) *shuffler {
	return &shuffler{
		rng: rand.New(rand.NewSource(rngSeed)),
	}
}

// shuffleBalls shuffles a slice of bingo balls in place using pseudo-random
// logic.
func (s *shuffler) shuffleBalls(balls []bingo.Ball) {
	for i := len(balls) - 1; i >= 1; i-- {
		randomIndex := s.rng.Intn(i + 1)
		elementToSwap := balls[i]
		balls[i] = balls[randomIndex]
		balls[randomIndex] = elementToSwap
	}
}
