// Package bingoshuffler defines a tool for shuffling bingo balls in
// pseudo-random, seed-based ways.
package shuffler

import (
	"math/rand"

	"github.com/Parkreiner/bingo"
)

// Shuffler provides methods for shuffling bingo calls using seed-based random
// logic.
type Shuffler struct {
	rng *rand.Rand
}

// NewShuffler creates a new instance of a Shuffler
func NewShuffler(rngSeed int64) *Shuffler {
	return &Shuffler{
		rng: rand.New(rand.NewSource(rngSeed)),
	}
}

// ShuffleBingoBalls shuffles a slice of bingo balls in place using
// pseudo-random logic.
func (s *Shuffler) ShuffleBingoBalls(balls []bingo.Ball) {
	for i := len(balls) - 1; i >= 1; i-- {
		randomIndex := s.rng.Intn(i + 1)
		elementToSwap := balls[i]
		balls[i] = balls[randomIndex]
		balls[randomIndex] = elementToSwap
	}
}
