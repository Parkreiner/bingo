package bingogen

import (
	"errors"
	"slices"
	"sync"

	"github.com/Parkreiner/bingo"
	"github.com/google/uuid"
)

// The number of cells that two cards are allowed to have in common to still
// be called unique from a fun, gameplay standpoint. A unique cell, in this
// case, refers to not just the numeric value of a cell, but also the
// position.
//
// The number is 2/3 of the total cells of a bingo card (25), excluding the
// free space.
const uniquenessThreshold = 16

type registryEntry struct {
	cells         [][]int
	id            uuid.UUID
	prevPlayerIDs []uuid.UUID
	active        bool
}

type cardRegistryStatus string

const (
	cardGenStatusIdle       cardRegistryStatus = "idle"
	cardGenStatusRunning    cardRegistryStatus = "running"
	cardGenStatusTerminated cardRegistryStatus = "terminated"
)

type CardRegistry struct {
	status            cardRegistryStatus
	statusMtx         *sync.RWMutex
	registeredEntries []*registryEntry
	entriesMtx        *sync.Mutex
	generator         *cellsGenerator
	doneChan          chan struct{}
	returnChan        chan *bingo.BingoCard
}

func NewCardRegistry(rngSeed int64) *CardRegistry {
	return &CardRegistry{
		status:            cardGenStatusIdle,
		registeredEntries: nil,
		entriesMtx:        &sync.Mutex{},
		statusMtx:         &sync.RWMutex{},
		generator:         newCardGenerator(rngSeed),
		doneChan:          make(chan struct{}),
		returnChan:        make(chan *bingo.BingoCard, 1),
	}
}

func (cg *CardRegistry) getStatus() cardRegistryStatus {
	cg.statusMtx.RLock()
	defer cg.statusMtx.Unlock()
	return cg.status
}

func (cg *CardRegistry) flushReturn(card *bingo.BingoCard) {
	if card == nil {
		return
	}

	cg.entriesMtx.Lock()
	defer cg.entriesMtx.Unlock()

	for _, entry := range cg.registeredEntries {
		if card.ID == entry.id {
			entry.active = false
			break
		}
	}
}

func (cg *CardRegistry) Start() (func(), error) {
	cleanup := func() {
		select {
		case cg.doneChan <- struct{}{}:
		default:
		}
	}

	status := cg.getStatus()
	if status == cardGenStatusTerminated {
		return nil, errors.New("trying to start terminated CardGen")
	}
	if status == cardGenStatusRunning {
		return cleanup, nil
	}

	cg.statusMtx.Lock()
	defer cg.statusMtx.Unlock()
	cg.status = cardGenStatusRunning

	go func() {
		defer func() {
			cg.statusMtx.Lock()
			defer cg.statusMtx.Unlock()
			cg.status = cardGenStatusTerminated
			close(cg.returnChan)
		}()

	loop:
		for {
			select {
			case <-cg.doneChan:
				break loop
			case returnedCard := <-cg.returnChan:
				cg.flushReturn(returnedCard)
			}
		}
	}()

	return cleanup, nil
}

func (cg *CardRegistry) generateUniqueEntry() *registryEntry {
	// Generating unique cards has a chance to take a while. Rather than let the
	// mutex stay locked the entire time, we can grab copies of the existing
	// cards and then unlock right after, so that other consumers can keep
	// accessing the registry while the generation is happening
	cg.entriesMtx.Lock()
	var cellSnapshots [][][]int
	for _, entry := range cg.registeredEntries {
		cellSnapshots = append(cellSnapshots, entry.cells)
	}
	cg.entriesMtx.Unlock()

	var newCells [][]int
	for {
		newCells = cg.generator.generateCells()
		cellConflicts := 0

		for _, snap := range cellSnapshots {
			for i, row := range snap {
				for j, cell := range row {
					// Skip over the free space
					if cell == -1 {
						continue
					}

					newCell := newCells[i][j]
					if cell == newCell {
						cellConflicts++
					}
				}
			}
		}

		if cellConflicts <= uniquenessThreshold {
			break
		}
	}

	return &registryEntry{
		cells:         newCells,
		id:            uuid.New(),
		prevPlayerIDs: nil,
		active:        false,
	}
}

func (cg *CardRegistry) checkOutRecycledEntry(playerID uuid.UUID) *registryEntry {
	cg.entriesMtx.Lock()
	defer cg.entriesMtx.Unlock()

	for _, entry := range cg.registeredEntries {
		foundReusable := !entry.active && !slices.Contains(entry.prevPlayerIDs, playerID)
		if foundReusable {
			entry.active = true
			entry.prevPlayerIDs = append(entry.prevPlayerIDs, playerID)
			return entry
		}
	}
	return nil
}

func (cg *CardRegistry) CheckOutCard(playerID uuid.UUID) (*bingo.BingoCard, error) {
	status := cg.getStatus()
	if status == cardGenStatusIdle {
		return nil, errors.New("must Start CardGen before calling other methods")
	}
	if status == cardGenStatusTerminated {
		return nil, errors.New("tried generating card for terminated CardGen")
	}

	var activeEntry *registryEntry
	reusable := cg.checkOutRecycledEntry(playerID)
	if reusable != nil {
		activeEntry = reusable
	} else {
		activeEntry = cg.generateUniqueEntry()

		cg.entriesMtx.Lock()
		activeEntry.active = true
		activeEntry.prevPlayerIDs = append(activeEntry.prevPlayerIDs, playerID)
		cg.registeredEntries = append(cg.registeredEntries, activeEntry)
		cg.entriesMtx.Unlock()
	}

	var statefulCells [][]*bingo.BingoCell
	for _, row := range activeEntry.cells {
		var statefulRow []*bingo.BingoCell
		for _, cell := range row {
			statefulRow = append(statefulRow, &bingo.BingoCell{
				Daubed: false,
				Value:  cell,
			})
		}
		statefulCells = append(statefulCells, statefulRow)
	}

	return &bingo.BingoCard{
		PlayerID: playerID,
		ID:       activeEntry.id,
		Cells:    statefulCells,
	}, nil
}

func (cg *CardRegistry) ReturnCard(card *bingo.BingoCard) error {
	status := cg.getStatus()
	if status == cardGenStatusIdle {
		return errors.New("must Start CardGen before returning card")
	}
	if status == cardGenStatusTerminated {
		return errors.New("tried returning card to terminated CardGen")
	}

	cg.returnChan <- card
	return nil
}
