package game

import (
	"fmt"
	"sync"
	"time"

	"github.com/Parkreiner/bingo"
	"github.com/google/uuid"
)

const maxSubscriberGoroutines = 100

type subscriptionEntry struct {
	id        uuid.UUID
	eventChan chan bingo.GameEvent
	phases    []bingo.GamePhase
}

type subscriptionsManager struct {
	subs []subscriptionEntry
	// Should always be buffered with some size
	routineBuffer chan struct{}
	mtx           *sync.Mutex
}

func newSubscriptionsManager() subscriptionsManager {
	buffer := make(chan struct{}, maxSubscriberGoroutines)
	for i := 0; i < maxSubscriberGoroutines; i++ {
		buffer <- struct{}{}
	}

	return subscriptionsManager{
		subs:          nil,
		routineBuffer: buffer,
		mtx:           &sync.Mutex{},
	}
}

func (sm *subscriptionsManager) dispatchEvent(event bingo.GameEvent) error {
	sm.mtx.Lock()
	maxBroadcasts := len(sm.subs)
	successfulBroadcasts := 0
	var subsCopy []subscriptionEntry
	for _, s := range sm.subs {
		subsCopy = append(subsCopy, s)
	}
	sm.mtx.Unlock()

	wg := sync.WaitGroup{}
	for _, s := range subsCopy {
		needToNotify := false
		for _, p := range s.phases {
			if p == event.Phase {
				needToNotify = true
				break
			}
		}
		if !needToNotify {
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
		return fmt.Errorf("dispatch failed for %d/%d subscribers", successfulBroadcasts, maxBroadcasts)
	}
	return nil
}

func (sm *subscriptionsManager) subscribe(phases []bingo.GamePhase) (<-chan bingo.GameEvent, func(), error) {
	sm.mtx.Lock()
	defer sm.mtx.Unlock()

	subID := uuid.New()
	eventChan := make(chan bingo.GameEvent, 1)
	sm.subs = append(sm.subs, subscriptionEntry{subID, eventChan, phases})

	subscribed := true
	unsubscribe := func() {
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
	}

	return eventChan, unsubscribe, nil
}
