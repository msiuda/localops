import type { HTMLAttributes } from "react";
import { cn } from "@/shared/lib/cn";

/** Monospaced inline text — paths, tool names, versions, variable names. */
export function MonoText({ className, ...props }: HTMLAttributes<HTMLSpanElement>) {
  return <span className={cn("font-mono", className)} {...props} />;
}
