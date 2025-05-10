package bingogen

import (
	"errors"
	"slices"
	"sync"
	"time"

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

const minEntrySurplus = 6 * bingo.MaxCards
const maxEntrySurplus = 20 * bingo.MaxCards

type registryEntry struct {
	// Should be treated as 100% immutable
	cells [][]int
	// Should be treated as 100% immutable
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
	surplusTicker     *time.Ticker
}

func NewCardRegistry(rngSeed int64) *CardRegistry {
	return &CardRegistry{
		status:            cardGenStatusIdle,
		registeredEntries: nil,
		entriesMtx:        &sync.Mutex{},
		statusMtx:         &sync.RWMutex{},
		generator:         newCellsGenerator(rngSeed),
		doneChan:          make(chan struct{}),
		returnChan:        make(chan *bingo.BingoCard, 1),
		surplusTicker:     nil,
	}
}

func (cg *CardRegistry) Status() cardRegistryStatus {
	cg.statusMtx.RLock()
	defer cg.statusMtx.Unlock()
	return cg.status
}

// This is meant to be a background operation, so we need to be very mindful of
// how long we keep things locked, especially since we'll be doing generation
// logic here
func (cg *CardRegistry) equalizeEntrySurplus() {
	// Add any needed surplus
	availableCards := 0
	for availableCards < minEntrySurplus {
		availableCards = 0

		cg.entriesMtx.Lock()
		for _, entry := range cg.registeredEntries {
			if !entry.active {
				availableCards++
			}
		}
		cg.entriesMtx.Unlock()

		_ = cg.generateUniqueEntry()
	}

	// Prune any extra surplus - there is a small risk that the value of
	// availableCards could get inaccurate by the time we do this if branch, but
	// that	shouldn't be a huge deal
	if availableCards < maxEntrySurplus {
		return
	}
	cg.entriesMtx.Lock()
	defer cg.entriesMtx.Unlock()
	slices.SortFunc(cg.registeredEntries, func(e1 *registryEntry, e2 *registryEntry) int {
		if e1.active && !e2.active {
			return -1
		}
		if e2.active && !e1.active {
			return 1
		}
		return 0
	})

	var endIndex int
	for endIndex = len(cg.registeredEntries) - 1; endIndex >= 0; endIndex-- {
		entry := cg.registeredEntries[endIndex]
		if entry.active {
			break
		}
	}
	cg.registeredEntries = cg.registeredEntries[0 : endIndex+1]
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
	status := cg.Status()
	if status == cardGenStatusTerminated {
		return nil, errors.New("trying to start terminated CardGen")
	}

	cleanup := func() {
		select {
		case cg.doneChan <- struct{}{}:
		default:
		}
	}
	if status == cardGenStatusRunning {
		return cleanup, nil
	}

	cg.statusMtx.Lock()
	defer cg.statusMtx.Unlock()
	cg.status = cardGenStatusRunning
	cg.equalizeEntrySurplus()
	cg.surplusTicker = time.NewTicker(5 * time.Second)

	go func() {
		defer func() {
			cg.statusMtx.Lock()
			defer cg.statusMtx.Unlock()
			cg.status = cardGenStatusTerminated
			close(cg.returnChan)
			cg.surplusTicker.Stop()
		}()

	loop:
		for {
			select {
			case <-cg.doneChan:
				break loop
			case returnedCard := <-cg.returnChan:
				cg.flushReturn(returnedCard)
			case <-cg.surplusTicker.C:
				cg.equalizeEntrySurplus()
			}
		}
	}()

	return cleanup, nil
}

func (cg *CardRegistry) generateUniqueEntry() *registryEntry {
	// Looked into trying to split up the unlocking logic, since there's a
	// chance that the uniqueness generation could take a while. That felt way
	// too risky, since even if we lock in two steps (once for grabbing
	// comparison snapshots, and again for appending the entry), there would be
	// a period of time when another consumer could generate a new card that
	// violates the uniqueness criteria of the card we just generated. Better to
	// stay locked the entire time to make race conditions impossible
	cg.entriesMtx.Lock()
	defer cg.entriesMtx.Unlock()
	var newCells [][]int
	for {
		newCells = cg.generator.generateCells()
		cellConflicts := 0

		for _, entry := range cg.registeredEntries {
			for i, row := range entry.cells {
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

	newEntry := &registryEntry{
		cells:         newCells,
		id:            uuid.New(),
		prevPlayerIDs: nil,
		active:        false,
	}
	cg.registeredEntries = append(cg.registeredEntries, newEntry)
	return newEntry
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
	status := cg.Status()
	if status == cardGenStatusIdle {
		return nil, errors.New("must Start CardGen before calling other methods")
	}
	if status == cardGenStatusTerminated {
		return nil, errors.New("tried generating card for terminated CardGen")
	}

	cg.entriesMtx.Lock()
	playerCards := 0
	for _, entry := range cg.registeredEntries {
		if slices.Contains(entry.prevPlayerIDs, playerID) {
			playerCards++
		}
	}
	cg.entriesMtx.Unlock()

	if playerCards >= bingo.MaxCards {
		return nil, errors.New("player cannot check out any more cards")
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
	status := cg.Status()
	if status == cardGenStatusIdle {
		return errors.New("must Start CardGen before returning card")
	}
	if status == cardGenStatusTerminated {
		return errors.New("tried returning card to terminated CardGen")
	}

	cg.returnChan <- card
	return nil
}
