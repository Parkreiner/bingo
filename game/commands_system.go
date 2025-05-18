package game

import (
	"fmt"

	"github.com/google/uuid"
)

func (g *Game) processSystemDispose(entityID uuid.UUID) error {
	if entityID != g.systemID {
		return fmt.Errorf("cannot fulfill system command for non-system. Received ID %q", entityID)
	}

	g.mtx.Lock()
	defer g.mtx.Unlock()

	var err error
	if g.dispose != nil {
		err = g.dispose()
	}
	return err
}

func (g *Game) processSystemBroadcastState(entityID uuid.UUID) error {
	if entityID != g.systemID {
		return fmt.Errorf("cannot fulfill system command for non-system. Received ID %q", entityID)
	}

	g.mtx.Lock()
	defer g.mtx.Unlock()

	return errTodo
}
