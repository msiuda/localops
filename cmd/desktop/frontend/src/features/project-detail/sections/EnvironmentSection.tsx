import type { EnvironmentSummary } from "@/entities/project";
import { SectionHeader } from "@/shared/ui/molecules/SectionHeader";
import { MonoText } from "@/shared/ui/atoms/MonoText";

interface EnvironmentSectionProps {
  environment: EnvironmentSummary;
}

const sourceContainer = "overflow-hidden rounded-md border border-border bg-surface";
const sourceRow =
  "flex items-center justify-between gap-4 px-3.5 py-2 [&:not(:last-child)]:border-b [&:not(:last-child)]:border-border";

/**
 * Renders the existing Environment Contract analysis as dense, aligned
 * data — a single structured list per section rather than one floating
 * card per source/variable, since a real project can declare dozens of
 * variables. Never renders a value — EnvironmentSummary is structurally
 * incapable of carrying one.
 */
export function EnvironmentSection({ environment }: EnvironmentSectionProps) {
  if (environment.error) {
    return <p className="text-danger text-micro m-0">Environment analysis failed: {environment.error}</p>;
  }

  if (!environment.hasContract) {
    return (
      <p className="text-text-muted text-micro m-0">
        No Environment Contract declared (.env.example, .env.sample, or .env.template).
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-5">
      <section className="flex flex-col gap-2">
        <SectionHeader>Contract sources</SectionHeader>
        <div className={sourceContainer}>
          {environment.contractSources?.map((source) => (
            <div key={source.file} className={sourceRow}>
              <MonoText className="text-text-primary text-emphasis">{source.file}</MonoText>
              <span className="text-text-muted text-micro flex-shrink-0">
                {source.variableCount} variable{source.variableCount === 1 ? "" : "s"}
              </span>
            </div>
          ))}
        </div>
      </section>

      {environment.localSources && environment.localSources.length > 0 && (
        <section className="flex flex-col gap-2">
          <SectionHeader>Local sources</SectionHeader>
          <div className={sourceContainer}>
            {environment.localSources.map((source) => (
              <div key={source.file} className={sourceRow}>
                <MonoText className="text-text-primary text-emphasis">{source.file}</MonoText>
                <span className="text-text-muted text-micro flex-shrink-0">
                  {source.variableCount} variable{source.variableCount === 1 ? "" : "s"}
                </span>
              </div>
            ))}
          </div>
        </section>
      )}

      <section className="flex flex-col gap-2">
        <SectionHeader>
          Variables
          {environment.variables ? ` (${environment.variables.length})` : ""}
        </SectionHeader>
        <div className="border-border bg-surface grid [grid-template-columns:minmax(160px,280px)_minmax(140px,200px)_1fr] overflow-hidden rounded-md border">
          {environment.variables?.map((variable, index) => (
            <div
              key={variable.name}
              className={`contents [&>*]:min-w-0 [&>*]:px-3.5 [&>*]:py-[7px] ${
                index < (environment.variables?.length ?? 0) - 1 ? "[&>*]:border-border [&>*]:border-b" : ""
              }`}
            >
              <MonoText
                className="text-text-primary text-emphasis truncate font-semibold"
                title={variable.name}
              >
                {variable.name}
              </MonoText>
              <span className="text-micro font-semibold whitespace-nowrap">
                {variable.satisfied ? (
                  <>
                    <span className="text-success">Satisfied</span>
                    <span className="text-text-muted font-normal"> · {variable.source}</span>
                  </>
                ) : (
                  <span className="text-warning">Missing</span>
                )}
              </span>
              <span className="text-text-muted text-micro truncate">{variable.declaredIn?.join(", ")}</span>
            </div>
          ))}
        </div>
      </section>

      {environment.findings && environment.findings.length > 0 && (
        <section className="flex flex-col gap-2">
          <SectionHeader>Findings</SectionHeader>
          <div className={sourceContainer}>
            {environment.findings.map((finding, index) => (
              <div key={index} className={sourceRow}>
                <MonoText className="text-text-muted text-micro flex-shrink-0">
                  {finding.source}
                  {finding.line > 0 ? `:${finding.line}` : ""}
                </MonoText>
                <span className="text-text-secondary text-body">{finding.detail}</span>
              </div>
            ))}
          </div>
        </section>
      )}
    </div>
  );
}
