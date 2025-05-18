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

// EventLogger handles logs of two types:
// 1. Automatic logs in response to every game event
// 2.
// Once instantiated, the logger will automatically start logging any events for
// phase types. The logger can be disposed by calling the Close method.
type EventLogger struct {
	file         *os.File
	loggerChan   chan loggerRequest
	disposedChan <-chan struct{}
}

var _ io.WriteCloser = &EventLogger{}

// Init is used to instantiate an EventFileLogger via the New function.
type Init struct {
	Subscriber bingo.PhaseSubscriber
	OutputPath string
}

// New instantiaes an EventFileLogger and automatically subscribes it to all
// events dispatched for every possible game event.
func New(init Init) (*EventLogger, error) {
	file, err := os.Open(init.OutputPath)
	if err != nil {
		return nil, fmt.Errorf("filepath %q does not exist: %v", init.OutputPath, err)
	}

	allEventsChan, unsub, err := init.Subscriber.Subscribe(nil)
	if err != nil {
		return nil, fmt.Errorf("unable to subscribe to all events: %v", err)
	}

	loggerChan := make(chan loggerRequest)
	disposedChan := make(chan struct{})
	logger := &EventLogger{
		file:         file,
		loggerChan:   loggerChan,
		disposedChan: disposedChan,
	}

	go func() {
		defer unsub()
		done := false

		for {
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
			case event, closed := <-allEventsChan:
				if closed {
					done = true
					break
				}
				logLine := fmt.Sprintf("[phase %s] [type %s] [id %s] %s", event.Phase, event.Type, event.ID, event.Message)
				_, _ = logger.file.Write([]byte(logLine))
			}

			if done {
				break
			}
		}

		close(disposedChan)
	}()

	return logger, nil
}

func (efl *EventLogger) Write(content []byte) (int, error) {
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
func (efl *EventLogger) Close() error {
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
