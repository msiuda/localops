import type { ButtonHTMLAttributes } from "react";
import { cn } from "@/shared/lib/cn";

export type ButtonVariant = "secondary" | "ghost";

const VARIANT_CLASSES: Record<ButtonVariant, string> = {
  secondary:
    "border border-border bg-surface text-text-secondary hover:enabled:bg-surface-hover hover:enabled:text-text-primary hover:enabled:border-border-strong",
  ghost: "text-text-secondary hover:enabled:bg-surface-hover hover:enabled:text-text-primary",
};

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
}

/**
 * The one reusable desktop-tool button primitive LocalOps currently needs:
 * compact, restrained, never a large call-to-action. Add variants here only
 * once a second genuinely different visual style is actually needed.
 */
export function Button({ variant = "secondary", className, type = "button", ...props }: ButtonProps) {
  return (
    <button
      type={type}
      className={cn(
        // Cursor (pointer when enabled, default when disabled) comes from
        // the base-layer `button` rules in app/styles/index.css — no need
        // to repeat it here as a utility.
        "inline-flex items-center gap-1.5 rounded-sm px-3 py-1.5 text-xs font-semibold",
        "disabled:opacity-60",
        "focus-visible:outline-accent focus-visible:outline-2 focus-visible:outline-offset-1",
        VARIANT_CLASSES[variant],
        className,
      )}
      {...props}
    />
  );
}
