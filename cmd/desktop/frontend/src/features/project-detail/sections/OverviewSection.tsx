import type { ProjectDetail } from "@/entities/project";
import { Badge } from "@/shared/ui/atoms/Badge";
import { SectionHeader } from "@/shared/ui/molecules/SectionHeader";

interface OverviewSectionProps {
  detail: ProjectDetail;
}

const group = "flex flex-col gap-2 rounded-md border border-border bg-surface p-3";
const row = "flex items-center justify-between gap-4";
const label = "text-body text-text-secondary";
const value = "text-body font-semibold text-text-primary";

/**
 * A compact "understand this project's local health in a few seconds"
 * summary — a two-column workspace (stack / diagnostics) at wide content
 * widths, collapsing to one column when narrow.
 */
export function OverviewSection({ detail }: OverviewSectionProps) {
  const env = detail.environment;

  return (
    <div className="grid grid-cols-1 items-start gap-5 min-[760px]:grid-cols-2">
      <section className={`${group} min-[760px]:row-span-2`}>
        <SectionHeader>Stack</SectionHeader>
        <div className={row}>
          <span className={label}>Technologies</span>
          {detail.technologies && detail.technologies.length > 0 ? (
            <div className="flex flex-wrap justify-end gap-1.5">
              {detail.technologies.map((tech) => (
                <Badge key={tech}>{tech}</Badge>
              ))}
            </div>
          ) : (
            <span className="text-text-muted text-micro">None detected</span>
          )}
        </div>
        <div className={row}>
          <span className={label}>Package manager</span>
          <span className={value}>{detail.packageManager || "—"}</span>
        </div>
      </section>

      <section className={group}>
        <SectionHeader>Doctor</SectionHeader>
        <div className={row}>
          <span className={label}>Issues</span>
          <span className={`${value} ${detail.issueCount > 0 ? "text-warning" : ""}`}>
            {detail.issueCount > 0
              ? `${detail.issueCount} issue${detail.issueCount === 1 ? "" : "s"}`
              : "None"}
          </span>
        </div>
      </section>

      <section className={group}>
        <SectionHeader>Environment</SectionHeader>
        {env.error ? (
          <p className="text-text-muted text-micro m-0">Environment analysis failed: {env.error}</p>
        ) : env.hasContract ? (
          <>
            <div className={row}>
              <span className={label}>Contract</span>
              <span className={value}>
                {env.contractSources?.length ?? 0} source{(env.contractSources?.length ?? 0) === 1 ? "" : "s"}
              </span>
            </div>
            <div className={row}>
              <span className={label}>Missing variables</span>
              <span className={`${value} ${env.missingCount > 0 ? "text-warning" : ""}`}>
                {env.missingCount > 0 ? env.missingCount : "None"}
              </span>
            </div>
          </>
        ) : (
          <p className="text-text-muted text-micro m-0">No Environment Contract declared.</p>
        )}
      </section>
    </div>
  );
}
