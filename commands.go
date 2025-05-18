package bingo

import (
	"encoding/json"
	"errors"

	"github.com/google/uuid"
)

// GameCommandType indicates what type of command is trying to be input into a
// ame. It can be used to know how what should be done in response to the
// incoming input, as well as know how the payload for the command is
// structured. The bingo package exports one custom struct for each command type
type GameCommandType = string

// ErrCommandNotSupported is used to indicate that a struct that implements the
// GameManager interface does NOT support a specific command.
var ErrCommandNotSupported = errors.New("command is not supported")

const (
	// GameCommandSystemBroadcastState instructs a game to broadcast the current
	// game state to a specified number of entities. Most useful for having the
	// game broadcast its state to the system, so that the system can sync the
	// state to all devices for the current player and the host
	GameCommandSystemBroadcastState GameCommandType = "system_broadcast_state"
	// GameCommandSystemDispose should be used to instruct a game to terminate
	// itself and tear down all resources. Once a game has been disposed, it is
	// assumed that it will not ever be updated, and it will automatically
	// handle all cleanup logic
	GameCommandSystemDispose GameCommandType = "system_dispose"
)

const (
	GameCommandHostStartGame            GameCommandType = "host_start_game"
	GameCommandHostTerminateGame        GameCommandType = "host_terminate_game"
	GameCommandHostBanPlayer            GameCommandType = "host_ban_player"
	GameCommandHostSuspendPlayer        GameCommandType = "host_suspend_player"
	GameCommandHostRequestBall          GameCommandType = "host_request_ball"
	GameCommandHostSyncBall             GameCommandType = "host_sync_ball"
	GameCommandHostAcknowledgeBingoCall GameCommandType = "host_acknowledge_bingo_call"
	GameCommandHostStartTiebreakerRound GameCommandType = "host_start_tiebreaker_round"
	// GameCommandHostAwardsPlayers indicates that the host acknowledges a
	// successful bingo call from one or more players. It is allowed to be
	// called at any time during the Confirming or Tiebreaker phases. For the
	// Tiebreaker phase specifically, it can be used to handle a tiebreaker
	// WITHOUT playing another round of bingo (i.e., making two players play
	// rock paper scissors to decide the winner). If a host is feeling generous,
	// they are allowed to award multiple players at once.
	GameCommandHostAwardsPlayers GameCommandType = "host_awards_players"
)

const (
	GameCommandPlayerDaub         GameCommandType = "player_daub"
	GameCommandPlayerUndoDaub     GameCommandType = "player_undo_daub"
	GameCommandPlayerCallBingo    GameCommandType = "player_call_bingo"
	GameCommandPlayerRescindBingo GameCommandType = "player_rescind_bingo"
	GameCommandPlayerReplaceCards GameCommandType = "player_replace_cards"
)

// GameCommand is any instruction that can be dispatched directly and
// synchronously to a game, by a "commander". A commander is currently defined
// as:
// 1. Players
// 2. Hosts
// 3. System (assumed to be whichever part of the app instantiated a game)
type GameCommand struct {
	// In TypeScript terms, Payload is any Record<string, unknown> type; it is
	// a JSON object that can contain any values. An accompanying struct type is
	// defined for each command type that needs a payload, so that you can parse
	// the payload with more type-safety. If an extra payload struct type
	// is *not* defined, you can assume the payload is empty/does not exist for
	// that command type
	Payload     json.RawMessage `json:"payload,omitempty"`
	Type        GameCommandType `json:"type"`
	CommanderID uuid.UUID       `json:"commanderId"`
}

type GameCommandPayloadSystemBroadcastState struct {
	// If the slice is nil/empty, it's assumed that the state should be
	// broadcast to all possible subscribers
	RecipientIDs []uuid.UUID `json:"recipientIds"`
}

type GameCommandPayloadTransferHostStatus struct {
	NewHostID []uuid.UUID `json:"newHostId"`
}

type GameCommandPayloadHostBanPlayer struct {
	PlayerID uuid.UUID `json:"playerId"`
}

type GameCommandPayloadHostSuspendPlayer struct {
	PlayerID uuid.UUID `json:"player_id"`
}

type GameCommandPayloadHostAwardsPlayers struct {
	// It is assumed that this field always has one element in it. If there are
	// no IDs, that results in an error.
	PlayerIDs []uuid.UUID `json:"playerIds"`
}

type GameCommandPayloadHostSyncBall struct {
	Value int `json:"value"`
}

type GameCommandPayloadPlayerDaub struct {
	CardID uuid.UUID `json:"cardId"`
	Cell   int       `json:"cell"`
}

type GameCommandPayloadPlayerUndoDaub struct {
	CardID uuid.UUID `json:"cardId"`
	Value  int       `json:"value"`
}
