import { Check } from "lucide-react";
import type { EnvironmentSummary, EnvUsage, EnvVariable } from "@/entities/project";
import { SectionHeader } from "@/shared/ui/molecules/SectionHeader";
import { MonoText } from "@/shared/ui/atoms/MonoText";

interface EnvironmentSectionProps {
  environment: EnvironmentSummary;
}

const sourceContainer = "overflow-hidden rounded-md border border-border bg-surface";
const sourceRow =
  "flex items-center justify-between gap-4 px-3.5 py-2 [&:not(:last-child)]:border-b [&:not(:last-child)]:border-border";

const th = "sticky top-0 z-10 bg-surface px-3.5 py-2 border-b border-border text-left";
const td = "px-3.5 py-[7px] align-top";

function usageTitle(usedIn: EnvUsage[]): string {
  return usedIn.map((u) => `${u.file}:${u.line}`).join(", ");
}

/** Primary usage location plus a compact count of the rest, so a variable used
 * across many files never explodes the row. A dash (with a tooltip explaining
 * why) replaces the old always-visible sentence for the common "declared, no
 * usage found" case — repeating that sentence down every row read as noise. */
function UsageCell({ variable }: { variable: EnvVariable }) {
  const usedIn = variable.usedIn;
  if (!variable.used || !usedIn || usedIn.length === 0) {
    return (
      <span className="text-text-muted text-micro" title="No supported static usage found">
        —
      </span>
    );
  }

  const [first, ...rest] = usedIn;
  return (
    <span className="text-micro" title={usageTitle(usedIn)}>
      <MonoText className="text-text-secondary">
        {first.file}:{first.line}
      </MonoText>
      {rest.length > 0 && <span className="text-text-muted"> +{rest.length} more</span>}
    </span>
  );
}

/** Whether the contract was consulted for this variable at all: Local is never
 * resolved for an undeclared name, which must read as "not evaluated," never
 * as "missing" (a claim LocalOps never checked and cannot make).
 *
 * ambiguous is true only when the column header itself names more than one
 * possible local source (e.g. ".env.local · .env", or a local file plus the
 * process environment) — only then does naming the specific satisfying
 * source on every row add information the header didn't already give. When
 * there is exactly one possible source, repeating its filename on every
 * healthy row is pure noise; the check mark alone, with the source in a
 * tooltip, says the same thing far more densely. */
function LocalCell({ variable, ambiguous }: { variable: EnvVariable; ambiguous: boolean }) {
  if (!variable.declared) {
    return (
      <span
        className="text-text-muted text-micro"
        title="Local sources are never evaluated for a variable the Environment Contract does not declare."
      >
        Not evaluated
      </span>
    );
  }

  if (variable.satisfied) {
    const sourceLabel = variable.source === "process environment" ? "process" : variable.source;
    if (!ambiguous) {
      return (
        <span title={`Satisfied by ${variable.source}`}>
          <Check size={13} strokeWidth={2.25} className="text-success" />
        </span>
      );
    }
    return (
      <span className="text-micro flex items-center gap-1 font-semibold whitespace-nowrap">
        <Check size={13} strokeWidth={2.25} className="text-success" />
        <span className="text-text-muted font-normal">{sourceLabel}</span>
      </span>
    );
  }

  return <span className="text-warning text-micro font-semibold whitespace-nowrap">Missing</span>;
}

/** The declared/undeclared contract fact. Visually distinct from Missing in
 * the Local column: Undeclared is a contract-authoring gap, Missing is a
 * local-environment gap, and conflating them would misdirect the fix. */
function ContractCell({ variable }: { variable: EnvVariable }) {
  if (!variable.declared) {
    return <span className="text-danger text-micro font-semibold whitespace-nowrap">Undeclared</span>;
  }

  return (
    <span className="text-micro flex items-center gap-1 font-semibold whitespace-nowrap">
      <Check size={13} strokeWidth={2.25} className="text-success" />
      {variable.declaredIn && variable.declaredIn.length > 1 && (
        <span className="text-text-muted truncate font-normal">{variable.declaredIn.join(", ")}</span>
      )}
    </span>
  );
}

/**
 * Renders the Environment Contract and source-usage analysis as a single
 * comparison matrix — Variable / Local / Contract / Usage — rather than
 * three separately-read facts the reader has to reassemble themselves. The
 * concrete source filenames live once, in the column headers, instead of
 * being repeated on every row. Never renders a value — EnvironmentSummary is
 * structurally incapable of carrying one.
 */
export function EnvironmentSection({ environment }: EnvironmentSectionProps) {
  if (environment.error) {
    return <p className="text-danger text-micro m-0">Environment analysis failed: {environment.error}</p>;
  }

  const variables = environment.variables ?? [];

  if (!environment.hasContract && variables.length === 0) {
    return (
      <p className="text-text-muted text-micro m-0">
        No Environment Contract declared (.env.example, .env.sample, or .env.template).
      </p>
    );
  }

  const localFiles = environment.localSources?.map((s) => s.file) ?? [];
  const usesProcessEnv = variables.some((v) => v.source === "process environment");
  const localLabel = [...localFiles, ...(usesProcessEnv ? ["process environment"] : [])].join(" · ");
  const contractLabel = environment.contractSources?.map((s) => s.file).join(" · ") ?? "";
  // Only worth naming the specific satisfying source per row when the
  // header itself names more than one possible local source.
  const ambiguousLocalSource = localFiles.length + (usesProcessEnv ? 1 : 0) > 1;

  return (
    <div className="flex flex-col gap-4">
      {!environment.hasContract && (
        <p className="text-text-muted text-micro m-0">
          No Environment Contract declared. Source usage was still detected below; local and process
          environment are not evaluated without a declared contract.
        </p>
      )}

      <section className="flex flex-col gap-2">
        <SectionHeader>Variables{variables.length > 0 ? ` (${variables.length})` : ""}</SectionHeader>
        <div className={`${sourceContainer} overflow-x-auto`}>
          <table className="w-full min-w-[640px] border-collapse">
            <colgroup>
              <col className="w-[34%]" />
              <col className="w-[16%]" />
              <col className="w-[16%]" />
              <col />
            </colgroup>
            <thead>
              <tr>
                <th className={`${th} text-text-muted text-micro font-bold tracking-[0.04em] uppercase`}>
                  Variable
                </th>
                <th className={`${th} text-text-muted text-micro font-bold tracking-[0.04em] uppercase`}>
                  <div className="flex flex-col gap-0.5">
                    <span>Local</span>
                    {localLabel && (
                      <MonoText
                        className="truncate font-normal tracking-normal normal-case"
                        title={localLabel}
                      >
                        {localLabel}
                      </MonoText>
                    )}
                  </div>
                </th>
                <th className={`${th} text-text-muted text-micro font-bold tracking-[0.04em] uppercase`}>
                  <div className="flex flex-col gap-0.5">
                    <span>Contract</span>
                    {contractLabel && (
                      <MonoText
                        className="truncate font-normal tracking-normal normal-case"
                        title={contractLabel}
                      >
                        {contractLabel}
                      </MonoText>
                    )}
                  </div>
                </th>
                <th className={`${th} text-text-muted text-micro font-bold tracking-[0.04em] uppercase`}>
                  Usage
                </th>
              </tr>
            </thead>
            <tbody>
              {variables.map((variable, index) => (
                <tr
                  key={variable.name}
                  className={index < variables.length - 1 ? "border-border border-b" : ""}
                >
                  <td className={td}>
                    <MonoText
                      className="text-text-primary text-emphasis block truncate font-semibold"
                      title={variable.name}
                    >
                      {variable.name}
                    </MonoText>
                  </td>
                  <td className={td}>
                    <LocalCell variable={variable} ambiguous={ambiguousLocalSource} />
                  </td>
                  <td className={td}>
                    <ContractCell variable={variable} />
                  </td>
                  <td className={`${td} truncate`}>
                    <UsageCell variable={variable} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
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
