import { Crown } from "lucide-react";
import type { FC } from "react";
import { cn } from "../../utils/styles";

type WinCountProps = Readonly<{
  children: number | string;
  className?: string;
}>;

export const WinCount: FC<WinCountProps> = ({ children: count, className }) => {
  return (
    <div
      className={cn(
        "flex max-w-fit flex-row items-center gap-2 rounded-md bg-slate-700 px-4 py-1 font-sans text-2xl leading-none font-extrabold text-slate-50",
        className,
      )}
    >
      <Crown className="relative top-[-1px]" strokeWidth={3} size={22} />
      <span aria-hidden>{count}</span>
      <span className="sr-only">You have won {count} times.</span>
    </div>
  );
};
