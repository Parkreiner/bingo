package game

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/Parkreiner/bingo"
)

type subscriptionsManager struct {
	// It is assumed that the map will be initialized with one entry per game
	// phase when a new game is instantiated
	subs map[bingo.GamePhase][]chan bingo.GameEvent
	mtx  *sync.Mutex
}

func newSubscriptionsManager() subscriptionsManager {
	subs := make(map[bingo.GamePhase][]chan bingo.GameEvent)
	for _, gp := range bingo.AllGamePhases {
		subs[gp] = nil
	}

	return subscriptionsManager{
		subs: subs,
		mtx:  &sync.Mutex{},
	}
}

func (sm *subscriptionsManager) pushEvent() error {
	return errTodo
}

// subscribeToPhaseEvents lets an external system subscribe to all events
// emitted during a given phase. There is no filtering beyond that – if the game
// is in the phase that was subscribed to, ALL events for all users will be
// emitted
func (sm *subscriptionsManager) subscribeToPhaseEvents(phase bingo.GamePhase) (<-chan bingo.GameEvent, func(), error) {
	sm.mtx.Lock()
	defer sm.mtx.Unlock()

	newEmitter := make(chan bingo.GameEvent)
	sm.subs[phase] = append(sm.subs[phase], newEmitter)

	subscribed := true
	unsubscribe := func() {
		if !subscribed {
			return
		}

		sm.mtx.Lock()
		defer sm.mtx.Unlock()

		var filtered []chan bingo.GameEvent
		for _, emitter := range sm.subs[phase] {
			if emitter != newEmitter {
				filtered = append(filtered, emitter)
			}
		}

		sm.subs[phase] = filtered
		close(newEmitter)
		subscribed = false
	}

	return newEmitter, unsubscribe, nil
}

// subscribeToAllEvents is a convenience method for subscribing to all
// possible phase events. It is fully equivalent to calling the
// SubscribeToPhaseEvents method once for each phase type, and then stitching
// the resulting return types together manually.
func (sm *subscriptionsManager) subscribeToAllEvents() (<-chan bingo.GameEvent, func(), error) {
	sm.mtx.Lock()
	defer sm.mtx.Unlock()

	var phaseEmitters []<-chan bingo.GameEvent
	var unsubCallbacks []func()
	for _, gp := range bingo.AllGamePhases {
		newEmitter, unsub, err := sm.subscribeToPhaseEvents(gp)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to subscribe for phase %q: %v", gp, err)
		}
		phaseEmitters = append(phaseEmitters, newEmitter)
		unsubCallbacks = append(unsubCallbacks, unsub)
	}

	consolidatedEmitter := make(chan bingo.GameEvent)
	go func() {
		var selectCases []reflect.SelectCase
		for _, pe := range phaseEmitters {
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
				if closeCount == len(phaseEmitters)-1 {
					break
				}
				continue
			}

			converted, ok := value.Interface().(bingo.GameEvent)
			if !ok {
				break
			}
			consolidatedEmitter <- converted
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
		close(consolidatedEmitter)
		subscribed = false
	}

	return consolidatedEmitter, unsubscribe, nil
}
