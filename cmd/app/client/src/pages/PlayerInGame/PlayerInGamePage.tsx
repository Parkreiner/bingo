import type { FC } from "react";
import { WinCount } from "./WinBadge";

export const PlayerInGamePage: FC = () => {
  return (
    <div className="mx-auto h-full max-w-5xl bg-red-500 px-4 pt-4">
      Blah
      <WinCount>{5}</WinCount>
    </div>
  );
};
