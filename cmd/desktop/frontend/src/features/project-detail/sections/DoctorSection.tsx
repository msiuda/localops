import type { DoctorCheck } from "@/entities/project";
import { MonoText } from "@/shared/ui/atoms/MonoText";

interface DoctorSectionProps {
  checks: DoctorCheck[];
}

/**
 * Renders the real Doctor report as-is; no logic recreated here.
 *
 * Each check is ONE real bordered row element (not borders scattered across
 * independent grid cells — that previously fragmented a single logical row
 * into three disconnected line segments). CSS subgrid lets every row's
 * internal tool/note/version columns still align with every other row's,
 * without resorting to `display: contents`.
 */
export function DoctorSection({ checks }: DoctorSectionProps) {
  if (checks.length === 0) {
    return <p className="text-text-muted m-0 text-xs">No Doctor checks apply to this project.</p>;
  }

  return (
    <div className="grid grid-cols-[minmax(150px,220px)_minmax(0,1fr)_max-content] gap-y-2">
      {checks.map((check) => (
        <div
          key={check.tool}
          className={`col-span-full grid grid-cols-subgrid items-center gap-x-4 rounded-md border px-4 py-2.5 ${
            check.ok ? "border-border" : "border-danger-bg bg-danger-bg"
          }`}
        >
          <div className="flex min-w-0 items-center gap-2.5">
            <MonoText className="text-text-primary text-[12.5px] font-semibold">{check.tool}</MonoText>
            <span
              className={`flex-shrink-0 text-[10.5px] font-bold tracking-[0.02em] uppercase ${
                check.ok ? "text-success" : "text-danger"
              }`}
            >
              {check.ok ? "Passed" : check.available ? "Failed" : "Not found"}
            </span>
          </div>
          <span className="text-text-secondary truncate text-xs">{check.note}</span>
          <MonoText className="text-text-muted text-right text-[11.5px] whitespace-nowrap">
            {check.version}
          </MonoText>
          {!check.ok && check.detail ? (
            <span className="text-danger col-span-full pt-1 text-[11.5px]">{check.detail}</span>
          ) : null}
        </div>
      ))}
    </div>
  );
}
