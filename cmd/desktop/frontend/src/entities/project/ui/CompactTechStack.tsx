import { Badge } from "@/shared/ui/atoms/Badge";

interface CompactTechStackProps {
  technologies?: string[] | null;
  moreTechnologies?: string[] | null;
  packageManager?: string;
  className?: string;
}

/**
 * The single, shared compact technology summary rendering, used by both
 * ProjectRow and the Project Detail header so neither independently
 * re-derives technology priority or overflow handling — the backend
 * already decided (see compactTechnologySummary), this only renders it.
 *
 * Always a single line: never wraps, so neither a row nor the header's
 * height ever depends on how many technologies were detected. A name
 * beyond the backend's cap never appears as a chip here — it is folded
 * into the trailing "+N" chip, whose title attribute lists the hidden
 * names. The first chip is always the primary language (the backend's
 * priority order guarantees this), so it alone reads slightly stronger —
 * everything else is one consistent, neutral chip style, never a rainbow
 * of per-technology colors. The package manager is deliberately not a
 * chip: it is metadata about the stack, not a technology identity, so it
 * renders as small muted text after a separator instead of competing
 * with the technology chips for visual weight.
 */
export function CompactTechStack({
  technologies,
  moreTechnologies,
  packageManager,
  className,
}: CompactTechStackProps) {
  const names = technologies ?? [];
  const more = moreTechnologies ?? [];

  if (names.length === 0 && !packageManager) {
    return null;
  }

  return (
    <div className={`flex min-w-0 flex-nowrap items-center gap-1.5 overflow-hidden ${className ?? ""}`}>
      {names.map((name, index) => (
        <Badge
          key={name}
          className={index === 0 ? "text-text-primary flex-shrink-0 font-semibold" : "flex-shrink-0"}
        >
          {name}
        </Badge>
      ))}
      {more.length > 0 && (
        <Badge className="flex-shrink-0" title={more.join(", ")}>
          +{more.length}
        </Badge>
      )}
      {packageManager && (
        <span className="text-text-muted text-micro flex-shrink-0 truncate whitespace-nowrap">
          · {packageManager}
        </span>
      )}
    </div>
  );
}
