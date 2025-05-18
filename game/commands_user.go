package game

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Parkreiner/bingo"
	"github.com/google/uuid"
)

func (g *Game) processPlayerDaub(command bingo.GameCommand) error {
	err := setDaubValue(g, command, true)

	var message string
	var eventType bingo.GameEventType
	if err != nil {
		message = "daubed card"
		eventType = bingo.EventTypeUpdate
	} else {
		message = "failed to daub card"
		eventType = bingo.EventTypeError
	}

	g.phaseSubscriptions.dispatchEvent(bingo.GameEvent{
		ID:           uuid.New(),
		Type:         eventType,
		CreatedByID:  command.CommanderID,
		Phase:        g.phase.value(),
		Message:      message,
		Created:      time.Now(),
		RecipientIDs: []uuid.UUID{command.CommanderID},
	})

	return err
}

func (g *Game) processPlayerUndoDaub(command bingo.GameCommand) error {
	err := setDaubValue(g, command, false)

	var message string
	var eventType bingo.GameEventType
	if err != nil {
		message = "removed daub from card"
		eventType = bingo.EventTypeUpdate
	} else {
		message = "failed to remove daub from card"
		eventType = bingo.EventTypeError
	}

	g.phaseSubscriptions.dispatchEvent(bingo.GameEvent{
		ID:           uuid.New(),
		Type:         eventType,
		CreatedByID:  command.CommanderID,
		Phase:        g.phase.value(),
		Message:      message,
		Created:      time.Now(),
		RecipientIDs: []uuid.UUID{command.CommanderID},
	})

	return err
}

func setDaubValue(game *Game, command bingo.GameCommand, daubValue bool) error {
	phase := game.phase.value()
	if phase == bingo.GamePhaseRoundStart {
		return errors.New("cannot change daubs when no cards have been called")
	}
	if phase == bingo.GamePhaseRoundEnd {
		return errors.New("phase is ending; daub change discarded")
	}

	game.mtx.Lock()
	defer game.mtx.Unlock()

	var player *bingo.Player
	for _, e := range game.cardPlayers {
		if e.player.ID == command.CommanderID {
			player = e.player
			break
		}
	}
	if player == nil {
		return fmt.Errorf("user with ID %q is not in game", command.CommanderID)
	}

	parsed := &bingo.GameCommandPayloadPlayerDaub{}
	if err := json.Unmarshal(command.Payload, parsed); err != nil {
		return fmt.Errorf("unable to parse daub payload: %v", err)
	}
	ball, err := bingo.ParseBall(parsed.Cell)
	if err != nil {
		return fmt.Errorf("%d is not a valid bingo ball", parsed.Cell)
	}

	var card *bingo.Card
	for _, c := range player.Cards {
		if c.ID == parsed.CardID {
			card = c
			break
		}
	}
	if card == nil {
		return fmt.Errorf("player %q does not have card with ID %q", player.Name, parsed.CardID)
	}

	// Actually daub the card - have to treat free space separately because it's
	// the one value that doesn't work with colIndex's math formula
	if ball == bingo.FreeSpace {
		cell := card.Cells[2][2]
		cell.Daubed = daubValue
		return nil
	}
	colIndex := (int(ball) - 1) / bingo.MaxBallValue
	var cell *bingo.Cell
	for i := 0; i < 5; i++ {
		c := card.Cells[i][colIndex]
		if c.Number == ball {
			cell = c
			break
		}
	}
	if cell == nil {
		return fmt.Errorf("value %d does not exist in card %q", ball, card.ID)
	}
	cell.Daubed = daubValue

	return nil
}
