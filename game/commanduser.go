package game

import (
	"encoding/json"
	"fmt"

	"github.com/Parkreiner/bingo"
)

// Daub value attempts to daub or un-daub a specific cell on a bingo card. If
// the cell that would be changed already has the daub value, that is treated as
// a no-op, and no error is returned. An error is only returned if the requested
// cell value does not exist in the card
func setDaubValue(card *bingo.Card, cellNum bingo.Ball, daubValue bool) error {
	colIndex := (cellNum - 1) / bingo.MaxBallValue

	var cell *bingo.Cell
	for i := 0; i < 5; i++ {
		c := card.Cells[i][colIndex]
		if c.Number == cellNum {
			cell = c
			break
		}
	}
	if cell == nil {
		return fmt.Errorf("value %d does not exist in card %q", cellNum, card.ID)
	}

	cell.Daubed = daubValue
	return nil
}

func (g *Game) processPlayerDaub(command bingo.GameCommand) error {
	g.mtx.Lock()
	defer g.mtx.Unlock()

	var player *bingo.Player
	for _, e := range g.cardPlayerEntries {
		if e.player.ID == command.CommanderEntityID {
			player = e.player
			break
		}
	}
	if player == nil {
		return fmt.Errorf("user with ID %q is not in game", command.CommanderEntityID)
	}

	parsed := &bingo.GameCommandPayloadPlayerDaub{}
	if err := json.Unmarshal(command.Payload, parsed); err != nil {
		return fmt.Errorf("unable to daub: %v", err)
	}
	ball, err := bingo.ParseBall(parsed.Value)
	if err != nil {
		return fmt.Errorf("%d is not a valid bingo ball", parsed.Value)
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

	return setDaubValue(card, ball, true)
}

func (g *Game) processPlayerUndoDaub(command bingo.GameCommand) error {
	g.mtx.Lock()
	defer g.mtx.Unlock()

	var player *bingo.Player
	for _, e := range g.cardPlayerEntries {
		if e.player.ID == command.CommanderEntityID {
			player = e.player
			break
		}
	}
	if player == nil {
		return fmt.Errorf("user with ID %q is not in game", command.CommanderEntityID)
	}

	parsed := &bingo.GameCommandPayloadPlayerDaub{}
	if err := json.Unmarshal(command.Payload, parsed); err != nil {
		return fmt.Errorf("unable to daub: %v", err)
	}
	ball, err := bingo.ParseBall(parsed.Value)
	if err != nil {
		return fmt.Errorf("%d is not a valid bingo ball", parsed.Value)
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

	return setDaubValue(card, ball, false)
}
