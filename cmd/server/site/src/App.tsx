import { useState } from "react";
import reactLogo from "./assets/react.svg";
import viteLogo from "/vite.svg";
import "./App.css";
import { generateUniqueBingoCards, MAX_CARDS, type BingoCard } from "./bingo";
import { useQuery } from "@tanstack/react-query";

type ServerResponse = Readonly<{
  sessionId: number;
  cards: readonly BingoCard[];
}>;

type Player = Readonly<{
  id: string;
  name: string;
}>;

type EventType = "game" | "error";

type Audience = "all" | "some" | "one";

type GameEvent = Readonly<{
  id: string;
  audience: Audience;
  type: EventType;
  message: string;
  recipientIds: string[];
}>;

type GamePhase =
  | "game_start"
  | "round_start"
  | "calling_balls"
  | "confirming_bingo"
  | "round_end"
  | "round_transition"
  | "game_over";

export type ServerRoom = Readonly<{
  id: string;
  joinCode: string;
  host: Player;
  round: number;
  phase: GamePhase;
  cards: readonly BingoCard[];
  players: readonly Player[];
  bingoCalls: readonly number[];
  events: readonly GameEvent[];

  winningPlayerIds: readonly string[];
  bingoCallerPlayerIds: readonly string[];
  suspendedPlayerIds: readonly string[];
  bannedPlayerIds: readonly string[];
  waitlistedPlayerIds: readonly string[];
}>;

export type PlayerRoom = Readonly<{
  id: string;
  joinCode: string;
  round: number;
  phase: GamePhase;
  cards: readonly BingoCard[];
  events: readonly string[];
}>;

function simulateServerCall(sessionId: number): Promise<ServerResponse> {
  return new Promise((resolve) => {
    window.setTimeout(() => {
      const cards = generateUniqueBingoCards(MAX_CARDS);
      resolve({ sessionId, cards });
    }, 1_500);
  });
}

function App() {
  const [sessionId, setSessionId] = useState(0);
  const query = useQuery({
    queryKey: [sessionId],
    queryFn: () => simulateServerCall(sessionId),
  });

  if (query.isLoading) {
    return <p>Loading&hellip;</p>;
  }

  return (
    <>
      <div>
        <a href="https://vite.dev" target="_blank" rel="noreferrer">
          <img src={viteLogo} className="logo" alt="Vite logo" />
        </a>
        <a href="https://react.dev" target="_blank" rel="noreferrer">
          <img src={reactLogo} className="logo react" alt="React logo" />
        </a>
      </div>
      <h1>Vite + React</h1>
      <div className="card">
        <button
          type="button"
          onClick={() => setSessionId((count) => count + 1)}
        >
          count is {sessionId}
        </button>
        <p>
          Edit <code>src/App.tsx</code> and save to test HMR
        </p>
      </div>
      <p className="read-the-docs">
        Click on the Vite and React logos to learn more
      </p>
    </>
  );
}

export default App;
