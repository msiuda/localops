import type { ReactNode } from "react";

/** A small uppercase group heading used inside dense diagnostic sections. */
export function SectionHeader({ children }: { children: ReactNode }) {
  return (
    <h2 className="text-text-muted m-0 text-[11px] font-bold tracking-[0.04em] uppercase">{children}</h2>
  );
}
