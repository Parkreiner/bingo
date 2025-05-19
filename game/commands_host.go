package game

import (
	"errors"
	"fmt"

	"github.com/Parkreiner/bingo"
	"github.com/google/uuid"
)

func (g *Game) processAutomaticBall(commanderID uuid.UUID) error {
	g.mtx.Lock()
	defer g.mtx.Unlock()

	if commanderID != g.host.ID {
		return fmt.Errorf("provided ID %q does not match host ID %q", commanderID, g.host.ID)
	}
	if g.phase.value() != bingo.GamePhaseCalling {
		return errors.New("can only issue a new ball during the calling phase")
	}

	ball, err := g.ballRegistry.NextAutomaticCall()
	if err != nil {
		g.phaseSubscriptions.dispatchEvent(bingo.GameEvent{
			Phase:        bingo.GamePhaseCalling,
			Type:         bingo.EventTypeError,
			CreatedByID:  commanderID,
			Message:      "unable to generate new ball",
			RecipientIDs: []uuid.UUID{commanderID},
		})
		return err
	}

	g.phaseSubscriptions.dispatchEvent(bingo.GameEvent{
		Phase:       bingo.GamePhaseCalling,
		Type:        bingo.EventTypeUpdate,
		CreatedByID: commanderID,
		Message:     fmt.Sprintf("new ball: %d", ball),

		// This one needs to be nil to make sure it reaches everyone
		RecipientIDs: nil,
	})
	return nil
}
