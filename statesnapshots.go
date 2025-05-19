package bingo

import "encoding/json"

// GameSnapshot is a snapshot of the current game state. It should be treated as
// a 100% immutable value.
// TODO: Figure out what other fields need to be on here
type GameSnapshot struct {
	Phase  GamePhase `json:"phase"`
	Called []Ball    `json:"called"`
}

var _ json.Marshaler = &GameSnapshot{}

// MarshalJSON takes a game snapshot, and serializes it as JSON. All nil slices
// will automatically be allocated to ensure they don't get serialized as JSON
// null.
func (gs *GameSnapshot) MarshalJSON() ([]byte, error) {
	snapCopy := GameSnapshot{
		Phase:  gs.Phase,
		Called: gs.Called,
	}
	if snapCopy.Called == nil {
		snapCopy.Called = []Ball{}
	}

	return json.Marshal(snapCopy)
}
