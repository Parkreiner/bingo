import { useLayoutEffect, useRef } from "react";

export type BingoCell = Readonly<{
  value: number;
  daubed: boolean;
}>;

export type BingoCard = Readonly<{
  id: string;
  playerId: string;
  cells: readonly (readonly BingoCell[])[];
}>;

export type PlayerSessionState = Readonly<{
  roomId: string;
  playerId: string;
  joinCode: string;
  phase: string;
  playerCount: number;
  winningPlayers: readonly string[];
  cards: readonly BingoCard[];
  suspension: null | Readonly<{
    playerId: string;
    duration: number;
    currentRound: number;
  }>;
}>;

export type GameCommand = Readonly<{
  type: string;
  commanderId: string;
  payload?: Record<string, unknown>;
}>;

export type UsePlayerSocketResult = Readonly<{
  state: PlayerSessionState;
  actions: Readonly<{
    daubCard: (cardId: string, cell: number) => void;
    unDaubCard: (cardId: string, cell: number) => void;
    callBingo: () => void;
    rescindBingo: () => void;
    replaceCards: () => void;
  }>;
}>;

export function usePlayerSocket(): UsePlayerSocketResult {
  // Normally wouldn't want to use useLayoutEffect for anything involving
  // network communication, but using it here ensures that all of the callbacks
  // that rely on the socket shouldn't ever have a period where the user can
  // interact with them while the socket is not initialized
  const socketRef = useRef<WebSocket>(null);
  useLayoutEffect(() => {
    const socket = new WebSocket("");
    socketRef.current = socket;

    return () => socket.close();
  }, []);

  const getSocket = (): WebSocket => {
    const socket = socketRef.current;
    if (socket === null) {
      throw new Error("Socket is not ready yet");
    }
    return socket;
  };

  const isDaubInputValid = (cardId: string, cell: number): boolean => {
    const isCellValid = Number.isInteger(cell) && cell >= 0 && cell <= 75;
    if (!isCellValid) {
      return false;
    }

    const foundCard = sessionState.cards.find((c) => c.id === cardId);
    if (foundCard === undefined) {
      return false;
    }

    if (cell === 0) {
      return true;
    }

    const colIndex = Math.floor((cell - 1) / 15);
    const matched = foundCard.cells.some(
      (row) => row[colIndex]?.value === cell,
    );
    return matched;
  };

  const sessionState: PlayerSessionState = {
    roomId: "1",
    playerId: "2",
    joinCode: "AAVC",
    phase: "calling",
    playerCount: 7,
    suspension: null,
    winningPlayers: ["1", "2", "2", "3", "3", "3", "4"],
    cards: [
      {
        id: "1",
        playerId: "2",
        cells: [
          [
            { value: 5, daubed: false },
            { value: 21, daubed: false },
            { value: 34, daubed: false },
            { value: 49, daubed: false },
            { value: 63, daubed: false },
          ],
          [
            { value: 4, daubed: false },
            { value: 20, daubed: false },
            { value: 33, daubed: false },
            { value: 48, daubed: false },
            { value: 62, daubed: false },
          ],
          [
            { value: 3, daubed: false },
            { value: 19, daubed: false },
            { value: 32, daubed: false },
            { value: 47, daubed: false },
            { value: 61, daubed: false },
          ],
          [
            { value: 2, daubed: false },
            { value: 18, daubed: false },
            { value: 31, daubed: false },
            { value: 46, daubed: false },
            { value: 75, daubed: false },
          ],
          [
            { value: 1, daubed: false },
            { value: 17, daubed: false },
            { value: 31, daubed: false },
            { value: 59, daubed: false },
            { value: 74, daubed: false },
          ],
        ],
      },
    ],
  };

  return {
    state: sessionState,
    actions: {
      daubCard: (cardId, cell) => {
        if (!isDaubInputValid(cardId, cell)) {
          return;
        }

        const command: GameCommand = {
          type: "player_daub",
          commanderId: sessionState.playerId,
          payload: { cardId, cell },
        };
        const socket = getSocket();
        socket.send(JSON.stringify(command));
      },

      unDaubCard: (cardId, cell) => {
        if (!isDaubInputValid(cardId, cell)) {
          return;
        }

        const command: GameCommand = {
          type: "player_undo_daub",
          commanderId: sessionState.playerId,
          payload: { cardId, cell },
        };
        const socket = getSocket();
        socket.send(JSON.stringify(command));
      },

      replaceCards: () => {
        const socket = getSocket();
        const command: GameCommand = {
          type: "player_replace_cards",
          commanderId: sessionState.playerId,
        };
        socket.send(JSON.stringify(command));
      },
      callBingo: () => {
        const socket = getSocket();
        const command: GameCommand = {
          type: "player_call_bingo",
          commanderId: sessionState.playerId,
        };
        socket.send(JSON.stringify(command));
      },
      rescindBingo: () => {
        const socket = getSocket();
        const command: GameCommand = {
          type: "player_rescind_bingo",
          commanderId: sessionState.playerId,
        };
        socket.send(JSON.stringify(command));
      },
    },
  };
}
