package game

import (
	"github.com/Parkreiner/bingo"
)

type cellsGenerator struct {
	shuffler *shuffler
}

func newCellsGenerator(seed int64) *cellsGenerator {
	return &cellsGenerator{
		shuffler: newShuffler(seed),
	}
}

func (cg *cellsGenerator) generateCells() [][]bingo.Ball {
	// Generate all cells. There might be a way to do this that doesn't involve
	// generating 10 extra cells per column, but the shuffling approach
	// guarantees that we cannot ever have duplicate cells in the same column
	allBCells := generateBingoBallsForRange(1, 15)
	allICells := generateBingoBallsForRange(16, 30)
	allNCells := generateBingoBallsForRange(31, 45)
	allGCells := generateBingoBallsForRange(46, 60)
	allOCells := generateBingoBallsForRange(61, 75)

	cg.shuffler.shuffleBalls(allBCells)
	cg.shuffler.shuffleBalls(allICells)
	cg.shuffler.shuffleBalls(allNCells)
	cg.shuffler.shuffleBalls(allGCells)
	cg.shuffler.shuffleBalls(allOCells)

	aggregateCells := [][]bingo.Ball{
		allBCells[0:5],
		allICells[0:5],
		allNCells[0:5],
		allGCells[0:5],
		allOCells[0:5],
	}
	aggregateCells[2][2] = bingo.FreeSpace

	// Rotate the card so that it looks like a proper bingo card, and so that
	// fewer data transformations need to be done per render in the frontend
	for i := 0; i < len(aggregateCells); i++ {
		row1 := aggregateCells[i]
		for j := i; j < len(row1); j++ {
			cell1 := row1[j]
			row2 := aggregateCells[j]
			cell2 := row2[i]
			row2[i] = cell1
			row1[j] = cell2
		}
	}

	return aggregateCells
}

// generateBingoBallsForRange creates a range of bingo balls for a given
// contiguous range. If the start or end bounds are invalid, the function will
// return a nil slice instead.
func generateBingoBallsForRange(start int, end int) []bingo.Ball {
	var cells []bingo.Ball
	inputIsInvalid := end <= start ||
		start <= 0 || end <= 0 ||
		start > bingo.MaxBallValue || end > bingo.MaxBallValue
	if inputIsInvalid {
		return cells
	}

	for i := start; i <= end; i++ {
		cells = append(cells, bingo.Ball(i))
	}
	return cells
}
