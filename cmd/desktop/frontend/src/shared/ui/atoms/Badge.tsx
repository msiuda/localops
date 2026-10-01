import type { ReactNode } from "react";
import { cn } from "@/shared/lib/cn";

interface BadgeProps {
  children: ReactNode;
  className?: string;
}

/** A small neutral label chip — e.g. a detected technology or package manager. */
export function Badge({ children, className }: BadgeProps) {
  return (
    <span
      className={cn(
        "border-border bg-surface text-text-secondary inline-flex items-center rounded-sm border px-[7px] py-0.5 text-[11px] font-medium whitespace-nowrap",
        className,
      )}
    >
      {children}
    </span>
  );
}
