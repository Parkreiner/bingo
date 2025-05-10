package bingogen

import (
	"math/rand"
)

type cellsGenerator struct {
	rng *rand.Rand
}

func newCardGenerator(seed int64) *cellsGenerator {
	return &cellsGenerator{
		rng: rand.New(rand.NewSource(seed)),
	}
}

func (cg *cellsGenerator) shuffleCellRange(cells []int) {
	for i := len(cells) - 1; i >= 1; i-- {
		randomIndex := cg.rng.Intn(i + 1)
		elementToSwap := cells[i]
		cells[i] = cells[randomIndex]
		cells[randomIndex] = elementToSwap
	}
}

func (cg *cellsGenerator) generateCellsForRange(start int, end int) []int {
	var cells []int
	for i := start; i <= end; i++ {
		cells = append(cells, i)
	}
	cg.shuffleCellRange(cells)
	return cells
}

func (cg *cellsGenerator) generateCells() [][]int {
	allBCells := cg.generateCellsForRange(1, 15)
	allICells := cg.generateCellsForRange(16, 30)
	allNCells := cg.generateCellsForRange(31, 45)
	allGCells := cg.generateCellsForRange(46, 60)
	allOCells := cg.generateCellsForRange(61, 75)

	aggregateCells := [][]int{
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
