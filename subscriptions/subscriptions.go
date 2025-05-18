// Package subscriptions makes it easy to manage subscriptions to game events.
package subscriptions

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/Parkreiner/bingo"
	"github.com/google/uuid"
)

const maxSubscriberGoroutines = 100

type subscriptionEntry struct {
	id             uuid.UUID
	eventChan      chan bingo.GameEvent
	filteredPhases []bingo.GamePhase
	recipientIDs   []uuid.UUID
	unsubscribe    func()
}

type Manager struct {
	subs []subscriptionEntry
	// Should always be buffered with some size
	routineBuffer chan struct{}
	// Should always be unbuffered
	disposedChan chan struct{}
	mtx          *sync.Mutex
}

func New() Manager {
	buffer := make(chan struct{}, maxSubscriberGoroutines)
	for i := 0; i < maxSubscriberGoroutines; i++ {
		buffer <- struct{}{}
	}

	return Manager{
		subs:          nil,
		routineBuffer: buffer,
		mtx:           &sync.Mutex{},
		disposedChan:  make(chan struct{}),
	}
}

func (sm *Manager) disposed() bool {
	sm.mtx.Lock()
	defer sm.mtx.Unlock()

	disposed := false
	select {
	case _, closed := <-sm.disposedChan:
		disposed = closed
	default:
	}

	return disposed
}

func (sm *Manager) DispatchEvent(event bingo.GameEvent) error {
	if sm.disposed() {
		return errors.New("not accepting new event dispatches")
	}

	sm.mtx.Lock()
	defer sm.mtx.Unlock()

	maxBroadcasts := len(sm.subs)
	successfulBroadcasts := 0
	wg := sync.WaitGroup{}

	for _, s := range sm.subs {
		if !isEligibleForDispatch(s, event) {
			continue
		}

		wg.Add(1)
		<-sm.routineBuffer
		go func() {
			defer func() {
				wg.Done()
				sm.routineBuffer <- struct{}{}
			}()

			select {
			case s.eventChan <- event:
				successfulBroadcasts++
			case <-time.After(2 * time.Second):
			}
		}()
	}
	wg.Wait()

	if successfulBroadcasts != maxBroadcasts {
		return fmt.Errorf("dispatch failed for %d/%d subscribers", maxBroadcasts-successfulBroadcasts, maxBroadcasts)
	}
	return nil
}

func (sm *Manager) Subscribe(phases []bingo.GamePhase, recipientIDs []uuid.UUID) (<-chan bingo.GameEvent, func(), error) {
	if sm.disposed() {
		return nil, nil, errors.New("not accepting new subscriptions")
	}

	sm.mtx.Lock()
	defer sm.mtx.Unlock()

	subID := uuid.New()
	eventChan := make(chan bingo.GameEvent, 1)
	subscribed := true

	entry := subscriptionEntry{
		id:             subID,
		eventChan:      eventChan,
		filteredPhases: phases,
		recipientIDs:   recipientIDs,
		unsubscribe: func() {
			if !subscribed {
				return
			}

			sm.mtx.Lock()
			defer sm.mtx.Unlock()

			var filtered []subscriptionEntry
			for _, entry := range sm.subs {
				if entry.id != subID {
					filtered = append(filtered, entry)
				}
			}

			sm.subs = filtered
			close(eventChan)
			subscribed = false
		},
	}

	sm.subs = append(sm.subs, entry)
	return eventChan, entry.unsubscribe, nil
}

func (sm *Manager) Dispose(systemID uuid.UUID) error {
	if sm.disposed() {
		return nil
	}

	err := sm.DispatchEvent(bingo.GameEvent{
		ID:           uuid.New(),
		Type:         bingo.EventTypeUpdate,
		Phase:        bingo.GamePhaseGameOver,
		CreatedByID:  systemID,
		Created:      time.Now(),
		RecipientIDs: nil,
		Message:      "Game has been terminated",
	})

	sm.mtx.Lock()
	var subsCopy []subscriptionEntry
	for _, s := range sm.subs {
		subsCopy = append(subsCopy, s)
	}
	// Have to close disposedChan here, because otherwise, there's a risk that
	// more subscribers will get added between locks/unlocks for the unsubscribe
	// callbacks
	close(sm.disposedChan)
	sm.mtx.Unlock()

	for _, s := range subsCopy {
		s.unsubscribe()
	}

	routinesCleared := 0
	for range sm.routineBuffer {
		routinesCleared++
		if routinesCleared == maxSubscriberGoroutines {
			break
		}
	}

	return err
}

func isEligibleForDispatch(subscription subscriptionEntry, event bingo.GameEvent) bool {
	matchesPhaseFilters := len(subscription.recipientIDs) == 0
	for _, p := range subscription.filteredPhases {
		if p == event.Phase {
			matchesPhaseFilters = true
			break
		}
	}
	if !matchesPhaseFilters {
		return false
	}

	recipientMatch := false
	for _, id := range event.RecipientIDs {
		if slices.Contains(subscription.recipientIDs, id) {
			recipientMatch = true
			break
		}
	}

	return recipientMatch
}
