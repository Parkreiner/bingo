package game

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

type subscriptionsManager struct {
	subs              []subscriptionEntry
	routineBufferSize int
	// Should always be buffered with some size
	routineBuffer chan struct{}
	// Should always be unbuffered
	disposedChan chan struct{}
	mtx          *sync.Mutex
}

func newSubscriptionsManager() subscriptionsManager {
	buffer := make(chan struct{}, maxSubscriberGoroutines)
	for i := 0; i < maxSubscriberGoroutines; i++ {
		buffer <- struct{}{}
	}

	return subscriptionsManager{
		subs:              nil,
		routineBuffer:     buffer,
		routineBufferSize: maxSubscriberGoroutines,
		mtx:               &sync.Mutex{},
		disposedChan:      make(chan struct{}),
	}
}

func (sm *subscriptionsManager) disposed() bool {
	sm.mtx.Lock()
	defer sm.mtx.Unlock()

	select {
	case _, closed := <-sm.disposedChan:
		return closed
	default:
	}

	return false
}

// dispatchUnsafe handles the core logic of dispatching events. It is NOT
// thread-safe; it is the rest of the struct's responsibility to call the method
// with any necessary thread protections.
func (sm *subscriptionsManager) dispatchUnsafe(event bingo.GameEvent) error {
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

	unfulfilled := maxBroadcasts - successfulBroadcasts
	if unfulfilled != 0 {
		return fmt.Errorf("dispatch failed for %d/%d subscribers", unfulfilled, maxBroadcasts)
	}
	return nil
}

// dispatchEvent notifies subscribers that an event has happened, using the
// event's fields to determine which subscribers need to be notified.
//
// It is safe to call this method without fully filling out an event. The
// following fields will be backfilled with data if they are a zero value:
// 1. Created - Backfilled with current time
// 2. ID - Backfilled with fresh UUID
//
// All other fields are assumed to be filled out with the correct data (which
// also means that the RecipientIDs field should only be nil if an event should
// be broadcast to all subscribers)
func (sm *subscriptionsManager) dispatchEvent(event bingo.GameEvent) error {
	if sm.disposed() {
		return errors.New("not accepting new event dispatches")
	}

	eventToDispatch := bingo.GameEvent{
		Created:      event.Created,
		ID:           event.ID,
		CreatedByID:  event.CreatedByID,
		Phase:        event.Phase,
		Message:      event.Message,
		Type:         event.Type,
		RecipientIDs: event.RecipientIDs,
	}
	if eventToDispatch.Created.IsZero() {
		eventToDispatch.Created = time.Now()
	}
	if eventToDispatch.ID == uuid.Nil {
		eventToDispatch.ID = uuid.New()
	}

	sm.mtx.Lock()
	defer sm.mtx.Unlock()
	return sm.dispatchUnsafe(eventToDispatch)
}

// subscribe lets an external system subscribe to events emitted by a game.
// Subscriptions can be "narrowed"/filtered by specifying a slice of game phases
// and a slice of recipients.
//
//   - If the phases slice is nil/empty, every eligible recipient will be
//     subscribed to ALL phases.
//   - If the recipients slice is nil/empty, EVERY subscriber will be notified
//     whenever an event is dispatched for a matching phase.
//   - If both slices are nil/empty, ALL subscribers will be subscribed to ALL
//     phases.
//
// The method returns a callback for manually unsubscribing. Note that:
//  1. The callback is safe to call multiple times.
//  2. The subscriptions manager can choose to unsubscribe a system even if that
//     system has never called the callback (mainly for teardown purposes).
//
// When the system has been unsubscribed (for any reason), the returned channel
// will automatically be closed.
func (sm *subscriptionsManager) subscribe(phases []bingo.GamePhase, recipientIDs []uuid.UUID) (<-chan bingo.GameEvent, func(), error) {
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

		// Need to define the core unsubscribe logic in a non-thread-safe way,
		// so that there's no deadlocking when trying to unsubscribe everything
		// as part of the dispose method. Just make sure to wrap thread-safety
		// around it before calling
		unsubscribe: func() {
			if !subscribed {
				return
			}

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

	safeUnsub := func() {
		sm.mtx.Lock()
		defer sm.mtx.Unlock()
		entry.unsubscribe()
	}
	return eventChan, safeUnsub, nil
}

// dispose cleans up a subscriptionsManager and renders it inert for any further
// event dispatches or subscription attempts. Calling it more than once results
// in a no-op.
func (sm *subscriptionsManager) dispose(systemID uuid.UUID) error {
	if sm.disposed() {
		return nil
	}

	sm.mtx.Lock()
	defer sm.mtx.Unlock()
	err := sm.dispatchUnsafe(bingo.GameEvent{
		ID:           uuid.New(),
		Type:         bingo.EventTypeUpdate,
		Phase:        bingo.GamePhaseGameOver,
		CreatedByID:  systemID,
		Created:      time.Now(),
		RecipientIDs: nil,
		Message:      "Game has been terminated",
	})

	for _, s := range sm.subs {
		s.unsubscribe()
	}

	routinesCleared := 0
	for range sm.routineBuffer {
		routinesCleared++
		if routinesCleared == sm.routineBufferSize {
			break
		}
	}

	// Considered also closing routineBuffer, but as long as all the methods
	// check disposedChan to see if they can do work, it should be safe to just
	// let that be garbage-collected
	close(sm.disposedChan)
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

	matchesRecipients := len(event.RecipientIDs) == 0
	for _, id := range event.RecipientIDs {
		if slices.Contains(subscription.recipientIDs, id) {
			matchesRecipients = true
			break
		}
	}

	return matchesRecipients
}
