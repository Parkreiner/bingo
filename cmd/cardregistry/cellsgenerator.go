package cardregistry

import (
	"math/rand"
)

type cellsGenerator struct {
	rng *rand.Rand
}

func newCellsGenerator(seed int64) *cellsGenerator {
	return &cellsGenerator{
		rng: rand.New(rand.NewSource(seed)),
	}
}

func (cg *cellsGenerator) shuffleCellRange(cells []int8) {
	for i := len(cells) - 1; i >= 1; i-- {
		randomIndex := cg.rng.Intn(i + 1)
		elementToSwap := cells[i]
		cells[i] = cells[randomIndex]
		cells[randomIndex] = elementToSwap
	}
}

func (cg *cellsGenerator) generateCells() [][]int8 {
	// Generate all cells. There might be a way to do this that doesn't involve
	// generating 10 extra cells per column, but the shuffling approach
	// guarantees that we cannot ever have duplicate cells in the same column
	allBCells := generateCellsForRange(1, 15)
	allICells := generateCellsForRange(16, 30)
	allNCells := generateCellsForRange(31, 45)
	allGCells := generateCellsForRange(46, 60)
	allOCells := generateCellsForRange(61, 75)

	cg.shuffleCellRange(allBCells)
	cg.shuffleCellRange(allICells)
	cg.shuffleCellRange(allNCells)
	cg.shuffleCellRange(allGCells)
	cg.shuffleCellRange(allOCells)

	// Slice off unneeded cells, and then swap in the free space
	aggregateCells := [][]int8{
		allBCells[0:5],
		allICells[0:5],
		allNCells[0:5],
		allGCells[0:5],
		allOCells[0:5],
	}
	aggregateCells[2][2] = -1

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

func generateCellsForRange(start int8, end int8) []int8 {
	var cells []int8
	if end <= start {
		return cells
	}

	for i := start; i <= end; i++ {
		cells = append(cells, i)
	}
	return cells
}
