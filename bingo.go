package bingo

import "github.com/google/uuid"

const (
	// The minimum number of cards a player is allowed to have in a game.
	MinCards = 1

	// The maximum number of cards a player is allowed to have in a game.
	MaxCards = 6
)

type BingoCell struct {
	// It is assumed that once a full card has been created, Value will remain
	// 100% static for as long as the card remains active in the game
	Value  int
	Daubed bool
}

type BingoCard struct {
	// A 5x5 grid of Bingo cells. Each column corresponds to a different "letter
	// "group" in the bingo board. That is:
	//
	// 1. Column 1 is column B and can have numbers 1–15
	// 2. Column 2 is column I and can have numbers 16–30
	// 3. Column 3 is column N and can have numbers 31–45, along with the free
	//   space in the middle
	// 4. Column 4 is column G and can have numbers 46–60
	// 5. Column 5 is column O and can have numbers 61–75
	//
	// The free space is represented as -1.
	Cells [][]*BingoCell

	ID       uuid.UUID
	PlayerID uuid.UUID
}
