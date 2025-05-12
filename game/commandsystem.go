package game

import (
	"fmt"

	"github.com/google/uuid"
)

func (g *Game) processSystemDispose(entityID uuid.UUID) error {
	g.mtx.Lock()
	defer g.mtx.Unlock()

	if entityID != g.systemID {
		return fmt.Errorf("cannot fulfill system command for non-system. Received ID %q", entityID)
	}

	if g.dispose != nil {
		g.dispose()
	}
	return nil
}

func (g *Game) processSystemBroadcastState(entityID uuid.UUID) error {
	g.mtx.Lock()
	defer g.mtx.Unlock()

	if entityID != g.systemID {
		return fmt.Errorf("cannot fulfill system command for non-system. Received ID %q", entityID)
	}

	return errTodo
}
