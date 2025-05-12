package game

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/Parkreiner/bingo"
)

const maxSubscriberGoroutines = 100

type subscriptionsManager struct {
	// It is assumed that the map will be initialized with one entry per game
	// phase when a new game is instantiated
	subs map[bingo.GamePhase][]chan bingo.GameEvent
	// Should always be buffered with some size
	routineBuffer chan struct{}
	mtx           *sync.Mutex
}

func newSubscriptionsManager() subscriptionsManager {
	subs := make(map[bingo.GamePhase][]chan bingo.GameEvent)
	for _, gp := range bingo.AllGamePhases {
		subs[gp] = nil
	}

	buffer := make(chan struct{}, maxSubscriberGoroutines)
	for i := 0; i < maxSubscriberGoroutines; i++ {
		buffer <- struct{}{}
	}

	return subscriptionsManager{
		subs:          subs,
		routineBuffer: buffer,
		mtx:           &sync.Mutex{},
	}
}

func (sm *subscriptionsManager) dispatchEvent(event bingo.GameEvent) error {
	sm.mtx.Lock()
	subs, ok := sm.subs[event.Phase]
	if !ok {
		return fmt.Errorf("received event with unknown phase %q", event.Phase)
	}

	maxBroadcasts := len(subs)
	successfulBroadcasts := 0
	var subsCopy []chan bingo.GameEvent
	for _, s := range subs {
		subsCopy = append(subsCopy, s)
	}
	sm.mtx.Unlock()

	wg := sync.WaitGroup{}
	for _, s := range subsCopy {
		wg.Add(1)
		<-sm.routineBuffer
		go func() {
			defer func() {
				wg.Done()
				sm.routineBuffer <- struct{}{}
			}()

			select {
			case s <- event:
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

// subscribeToPhaseEvents lets an external system subscribe to all events
// emitted during a given phase. There is no filtering beyond that – if the game
// is in the phase that was subscribed to, ALL events for all users will be
// emitted
func (sm *subscriptionsManager) subscribeToPhaseEvents(phase bingo.GamePhase) (<-chan bingo.GameEvent, func(), error) {
	sm.mtx.Lock()
	defer sm.mtx.Unlock()

	newChan := make(chan bingo.GameEvent)
	sm.subs[phase] = append(sm.subs[phase], newChan)

	subscribed := true
	unsubscribe := func() {
		if !subscribed {
			return
		}

		sm.mtx.Lock()
		defer sm.mtx.Unlock()

		var filtered []chan bingo.GameEvent
		for _, eventC := range sm.subs[phase] {
			if eventC != newChan {
				filtered = append(filtered, eventC)
			}
		}

		sm.subs[phase] = filtered
		close(newChan)
		subscribed = false
	}

	return newChan, unsubscribe, nil
}

// subscribeToAllEvents is a convenience method for subscribing to all
// possible phase events. It is fully equivalent to calling the
// SubscribeToPhaseEvents method once for each phase type, and then stitching
// the resulting return types together manually.
func (sm *subscriptionsManager) subscribeToAllEvents() (<-chan bingo.GameEvent, func(), error) {
	sm.mtx.Lock()
	defer sm.mtx.Unlock()

	var phaseChans []<-chan bingo.GameEvent
	var unsubCallbacks []func()
	for _, gp := range bingo.AllGamePhases {
		newChan, unsub, err := sm.subscribeToPhaseEvents(gp)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to subscribe for phase %q: %v", gp, err)
		}
		phaseChans = append(phaseChans, newChan)
		unsubCallbacks = append(unsubCallbacks, unsub)
	}

	consolidated := make(chan bingo.GameEvent)
	go func() {
		var selectCases []reflect.SelectCase
		for _, pe := range phaseChans {
			selectCases = append(selectCases, reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(pe),
			})
		}

		closeCount := 0
		for {
			_, value, closed := reflect.Select(selectCases)
			if closed {
				closeCount++
				if closeCount == len(phaseChans)-1 {
					break
				}
				continue
			}

			converted, ok := value.Interface().(bingo.GameEvent)
			if !ok {
				break
			}
			consolidated <- converted
		}
	}()

	subscribed := true
	unsubscribe := func() {
		if !subscribed {
			return
		}

		sm.mtx.Lock()
		defer sm.mtx.Unlock()

		for _, cb := range unsubCallbacks {
			cb()
		}
		close(consolidated)
		subscribed = false
	}

	return consolidated, unsubscribe, nil
}
