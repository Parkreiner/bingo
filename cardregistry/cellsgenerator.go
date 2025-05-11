package cardregistry

import (
	"github.com/Parkreiner/bingo"
	"github.com/Parkreiner/bingo/shuffler"
)

type cellsGenerator struct {
	shuffler *shuffler.Shuffler
}

func newCellsGenerator(seed int64) *cellsGenerator {
	return &cellsGenerator{
		shuffler: shuffler.NewShuffler(seed),
	}
}

func (cg *cellsGenerator) generateCells() [][]bingo.Ball {
	// Generate all cells. There might be a way to do this that doesn't involve
	// generating 10 extra cells per column, but the shuffling approach
	// guarantees that we cannot ever have duplicate cells in the same column
	allBCells := bingo.GenerateBingoBallsForRange(1, 15)
	allICells := bingo.GenerateBingoBallsForRange(16, 30)
	allNCells := bingo.GenerateBingoBallsForRange(31, 45)
	allGCells := bingo.GenerateBingoBallsForRange(46, 60)
	allOCells := bingo.GenerateBingoBallsForRange(61, 75)

	cg.shuffler.ShuffleBingoBalls(allBCells)
	cg.shuffler.ShuffleBingoBalls(allICells)
	cg.shuffler.ShuffleBingoBalls(allNCells)
	cg.shuffler.ShuffleBingoBalls(allGCells)
	cg.shuffler.ShuffleBingoBalls(allOCells)

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
