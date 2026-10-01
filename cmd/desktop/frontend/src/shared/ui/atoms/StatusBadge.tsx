import type { LucideIcon } from "lucide-react";
import { cn } from "@/shared/lib/cn";

export type StatusTone = "success" | "warning" | "danger" | "muted";

const TONE_CLASSES: Record<StatusTone, string> = {
  success: "text-success bg-success-bg",
  warning: "text-warning bg-warning-bg",
  danger: "text-danger bg-danger-bg",
  muted: "text-text-muted bg-muted-bg",
};

interface StatusBadgeProps {
  label: string;
  tone: StatusTone;
  icon?: LucideIcon;
  className?: string;
}

/** A small pill communicating a status (health, pass/fail, etc.) via tone + label. */
export function StatusBadge({ label, tone, icon: Icon, className }: StatusBadgeProps) {
  return (
    <span
      className={cn(
        "text-micro inline-flex items-center gap-[5px] rounded-full px-2 py-0.5 font-semibold tracking-[0.01em] whitespace-nowrap",
        TONE_CLASSES[tone],
        className,
      )}
    >
      {Icon ? <Icon size={12} strokeWidth={2.5} /> : null}
      {label}
    </span>
  );
}
