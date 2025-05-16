// Package eventfilelogger provides an easy way to write logs describing game
// events to a specific file.
package eventfilelogger

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/Parkreiner/bingo"
)

type logWriteResult struct {
	bytesWritten int
	err          error
}

type loggerRequest struct {
	content    []byte
	resultChan chan<- logWriteResult
}

// EventFileLogger handles logs of two types:
// 1. Automatic logs in response to every game event
// 2.
// Once instantiated, the logger will automatically start logging any events for
// phase types. The logger can be disposed by calling the Close method.
type EventFileLogger struct {
	file         *os.File
	loggerChan   chan loggerRequest
	disposedChan <-chan struct{}
}

var _ io.WriteCloser = &EventFileLogger{}

// Init is used to instantiate an EventFileLogger via the New function.
type Init struct {
	Subscriber bingo.PhaseSubscriber
	OutputPath string
}

// New instantiaes an EventFileLogger and automatically subscribes it to all
// events dispatched for every possible game event.
func New(init Init) (*EventFileLogger, error) {
	file, err := os.Open(init.OutputPath)
	if err != nil {
		return nil, fmt.Errorf("filepath %q does not exist: %v", init.OutputPath, err)
	}

	// Set up subscriptions for each phase type (making sure to close any
	// subscriptions if any fail). As annoying as setting this all up manually
	// is, trying to stitch everything together with reflection will tank
	// performance a lot
	var unsubCallbacks []func()
	unsubToAll := func() {
		for _, unsub := range unsubCallbacks {
			unsub()
		}
	}

	initChan, initUnsub, err := init.Subscriber.SubscribeToPhaseEvents(bingo.GamePhaseInitialized)
	if err != nil {
		unsubToAll()
		return nil, fmt.Errorf("unable to subscribe to events for phase %s", bingo.GamePhaseInitialized)
	}
	unsubCallbacks = append(unsubCallbacks, initUnsub)

	roundStartChan, roundStartUnsub, err := init.Subscriber.SubscribeToPhaseEvents(bingo.GamePhaseRoundStart)
	if err != nil {
		unsubToAll()
		return nil, fmt.Errorf("unable to subscribe to events for phase %s", bingo.GamePhaseRoundStart)
	}
	unsubCallbacks = append(unsubCallbacks, roundStartUnsub)

	callingChan, callingUnsub, err := init.Subscriber.SubscribeToPhaseEvents(bingo.GamePhaseCalling)
	if err != nil {
		unsubToAll()
		return nil, fmt.Errorf("unable to subscribe to events for phase %s", bingo.GamePhaseCalling)
	}
	unsubCallbacks = append(unsubCallbacks, callingUnsub)

	confirmingChan, confirmingUnsub, err := init.Subscriber.SubscribeToPhaseEvents(bingo.GamePhaseConfirmingBingo)
	if err != nil {
		unsubToAll()
		return nil, fmt.Errorf("unable to subscribe to events for phase %s", bingo.GamePhaseConfirmingBingo)
	}
	unsubCallbacks = append(unsubCallbacks, confirmingUnsub)

	tiebreakerChan, tiebreakerUnsub, err := init.Subscriber.SubscribeToPhaseEvents(bingo.GamePhaseTiebreaker)
	if err != nil {
		unsubToAll()
		return nil, fmt.Errorf("unable to subscribe to events for phase %s", bingo.GamePhaseTiebreaker)
	}
	unsubCallbacks = append(unsubCallbacks, tiebreakerUnsub)

	roundEndChan, roundEndUnsub, err := init.Subscriber.SubscribeToPhaseEvents(bingo.GamePhaseRoundEnd)
	if err != nil {
		unsubToAll()
		return nil, fmt.Errorf("unable to subscribe to events for phase %s", bingo.GamePhaseRoundEnd)
	}
	unsubCallbacks = append(unsubCallbacks, roundEndUnsub)

	gameOverChan, gameOverUnsub, err := init.Subscriber.SubscribeToPhaseEvents(bingo.GamePhaseGameOver)
	if err != nil {
		unsubToAll()
		return nil, fmt.Errorf("unable to subscribe to events for phase %s", bingo.GamePhaseGameOver)
	}
	unsubCallbacks = append(unsubCallbacks, gameOverUnsub)

	loggerChan := make(chan loggerRequest)
	disposedChan := make(chan struct{})
	logger := &EventFileLogger{
		file:         file,
		loggerChan:   loggerChan,
		disposedChan: disposedChan,
	}

	go func() {
		defer unsubToAll()
		done := false

		for {
			var event *bingo.GameEvent
			select {
			case req, closed := <-loggerChan:
				if closed {
					done = true
					break
				}
				b, err := logger.file.Write(req.content)
				req.resultChan <- logWriteResult{
					bytesWritten: b,
					err:          err,
				}

			case e, closed := <-initChan:
				if closed {
					break
				}
				event = &e
			case e, closed := <-roundStartChan:
				if closed {
					break
				}
				event = &e
			case e, closed := <-callingChan:
				if closed {
					break
				}
				event = &e
			case e, closed := <-confirmingChan:
				if closed {
					break
				}
				event = &e
			case e, closed := <-tiebreakerChan:
				if closed {
					break
				}
				event = &e
			case e, closed := <-roundEndChan:
				if closed {
					break
				}
				event = &e
			case e, closed := <-gameOverChan:
				if closed {
					break
				}
				event = &e
			}

			if done {
				break
			}
			if event != nil {
				logger.writeEventToFile(*event)
			}
		}

		close(disposedChan)
	}()

	return logger, nil
}

func (efl *EventFileLogger) writeEventToFile(event bingo.GameEvent) error {
	logLine := fmt.Sprintf("[phase %s] [type %s] [id %s] %s", event.Phase, event.Type, event.ID, event.Message)
	_, err := efl.file.Write([]byte(logLine))
	if err != nil {
		return fmt.Errorf("unable to write log %q: %v", logLine, err)
	}
	return nil
}

func (efl *EventFileLogger) Write(content []byte) (int, error) {
	select {
	case _, closed := <-efl.disposedChan:
		if closed {
			return 0, errors.New("logger is closed")
		}
	default:
	}

	resultChan := make(chan logWriteResult)
	efl.loggerChan <- loggerRequest{
		content:    content,
		resultChan: resultChan,
	}

	result := <-resultChan
	return result.bytesWritten, result.err
}

// Close terminates an EventFileLogger, rendering it so that it can no longer
// receive logs. It will also close all open subscriptions. This function is
// safe to call multiple times; calling it more than once results in a no-op.
func (efl *EventFileLogger) Close() error {
	select {
	case _, closed := <-efl.disposedChan:
		if closed {
			return nil
		}
	default:
	}

	close(efl.loggerChan)
	<-efl.disposedChan
	return nil
}
