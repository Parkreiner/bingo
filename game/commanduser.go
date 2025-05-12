package game

import (
	"encoding/json"
	"fmt"

	"github.com/Parkreiner/bingo"
)

func setDaubValue(game *Game, command bingo.GameCommand, daubValue bool) error {
	game.mtx.Lock()
	defer game.mtx.Unlock()

	var player *bingo.Player
	for _, e := range game.cardPlayerEntries {
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
		return fmt.Errorf("unable to parse daub payload: %v", err)
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

	// Actually daub the card - have to treat free space separately because it's
	// the one value that doesn't work with colIndex's math formula
	if ball == bingo.FreeSpace {
		cell := card.Cells[2][2]
		cell.Daubed = daubValue
		return nil
	}
	colIndex := (ball - 1) / bingo.Ball(bingo.MaxBallValue)
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

func (g *Game) processPlayerDaub(command bingo.GameCommand) error {
	return setDaubValue(g, command, true)
}

func (g *Game) processPlayerUndoDaub(command bingo.GameCommand) error {
	return setDaubValue(g, command, false)
}
