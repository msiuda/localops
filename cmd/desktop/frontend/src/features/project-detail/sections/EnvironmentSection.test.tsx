import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { EnvironmentSection } from "./EnvironmentSection";
import type { EnvironmentSummary, EnvVariable } from "@/entities/project";

function summary(overrides: Partial<EnvironmentSummary>): EnvironmentSummary {
  return {
    hasContract: true,
    contractSources: [],
    localSources: [],
    variables: [],
    findings: [],
    missingCount: 0,
    undeclaredCount: 0,
    error: "",
    ...overrides,
  };
}

function variable(overrides: Partial<EnvVariable>): EnvVariable {
  return {
    name: "VAR",
    declared: true,
    declaredIn: [],
    used: false,
    usedIn: [],
    satisfied: false,
    source: "",
    ...overrides,
  };
}

describe("EnvironmentSection", () => {
  it("shows a healthy declared + present + used variable", () => {
    render(
      <EnvironmentSection
        environment={summary({
          contractSources: [{ file: ".env.example", variableCount: 1 }],
          localSources: [{ file: ".env.local", variableCount: 1 }],
          variables: [
            variable({
              name: "DATABASE_URL",
              declaredIn: [".env.example"],
              used: true,
              usedIn: [{ file: "src/db.ts", line: 14 }],
              satisfied: true,
              source: ".env.local",
            }),
          ],
        })}
      />,
    );

    expect(screen.getByText("DATABASE_URL")).toBeInTheDocument();
    expect(screen.getByText("src/db.ts:14")).toBeInTheDocument();
    expect(screen.getAllByText(".env.local").length).toBeGreaterThan(0);
    expect(screen.queryByText("Missing")).not.toBeInTheDocument();
    expect(screen.queryByText("Undeclared")).not.toBeInTheDocument();

    // Never a value, only names/status/sources.
    expect(document.body.textContent).not.toMatch(/postgres:\/\//);
  });

  it("shows a declared + missing + used variable", () => {
    render(
      <EnvironmentSection
        environment={summary({
          contractSources: [{ file: ".env.example", variableCount: 1 }],
          variables: [
            variable({
              name: "JWT_SECRET",
              declaredIn: [".env.example"],
              used: true,
              usedIn: [{ file: "src/auth.ts", line: 9 }],
              satisfied: false,
            }),
          ],
        })}
      />,
    );

    expect(screen.getByText("Missing")).toBeInTheDocument();
    expect(screen.getByText("src/auth.ts:9")).toBeInTheDocument();
  });

  it("marks a used-but-undeclared variable Undeclared with Local read as Not evaluated", () => {
    render(
      <EnvironmentSection
        environment={summary({
          hasContract: true,
          contractSources: [{ file: ".env.example", variableCount: 1 }],
          variables: [
            variable({
              name: "STRIPE_SECRET_KEY",
              declared: false,
              declaredIn: [],
              used: true,
              usedIn: [{ file: "src/payments/stripe.ts", line: 21 }],
            }),
          ],
          undeclaredCount: 1,
        })}
      />,
    );

    expect(screen.getByText("STRIPE_SECRET_KEY")).toBeInTheDocument();
    expect(screen.getByText("Undeclared")).toBeInTheDocument();
    expect(screen.getByText("Not evaluated")).toBeInTheDocument();
    expect(screen.queryByText("Missing")).not.toBeInTheDocument();
  });

  it("renders both facts when there is no contract but source usage was detected", () => {
    render(
      <EnvironmentSection
        environment={summary({
          hasContract: false,
          contractSources: [],
          localSources: [],
          variables: [
            variable({
              name: "DATABASE_URL",
              declared: false,
              declaredIn: [],
              used: true,
              usedIn: [{ file: "src/db.ts", line: 12 }],
            }),
          ],
          undeclaredCount: 1,
        })}
      />,
    );

    expect(screen.getByText(/no environment contract declared/i)).toBeInTheDocument();
    expect(screen.getByText("DATABASE_URL")).toBeInTheDocument();
    expect(screen.getByText("Undeclared")).toBeInTheDocument();
    expect(screen.getByText("Not evaluated")).toBeInTheDocument();
    expect(screen.getByText("src/db.ts:12")).toBeInTheDocument();
  });

  it("uses a dense dash, never 'unused', for a declared variable with no supported usage found", () => {
    render(
      <EnvironmentSection
        environment={summary({
          variables: [
            variable({
              name: "LEGACY_FEATURE",
              declaredIn: [".env.example"],
              satisfied: true,
              source: ".env",
            }),
          ],
        })}
      />,
    );

    const usageDash = screen.getByTitle("No supported static usage found");
    expect(usageDash).toHaveTextContent("—");
    expect(screen.queryByText(/unused/i)).not.toBeInTheDocument();
  });

  it("collapses multiple usage locations into a compact count instead of exploding the row", () => {
    render(
      <EnvironmentSection
        environment={summary({
          variables: [
            variable({
              name: "DATABASE_URL",
              declaredIn: [".env.example"],
              used: true,
              usedIn: [
                { file: "src/a.ts", line: 1 },
                { file: "src/b.ts", line: 2 },
                { file: "src/c.ts", line: 3 },
              ],
              satisfied: true,
              source: ".env",
            }),
          ],
        })}
      />,
    );

    expect(screen.getByText("src/a.ts:1")).toBeInTheDocument();
    expect(screen.getByText("+2 more")).toBeInTheDocument();
    expect(screen.queryByText("src/b.ts:2")).not.toBeInTheDocument();
  });

  it("does not repeat the filename per row when exactly one local source exists", () => {
    render(
      <EnvironmentSection
        environment={summary({
          contractSources: [{ file: ".env.example", variableCount: 1 }],
          localSources: [{ file: ".env", variableCount: 1 }],
          variables: [
            variable({ name: "DATABASE_URL", declaredIn: [".env.example"], satisfied: true, source: ".env" }),
          ],
        })}
      />,
    );

    // The header names the single source; the row itself must not repeat
    // it — only the tooltip does.
    expect(screen.getByTitle("Satisfied by .env")).toBeInTheDocument();
    const cells = screen.getAllByText(".env");
    // The only visible ".env" text is in the column header, not the row.
    expect(cells).toHaveLength(1);
  });

  it("shows the specific satisfying source per row when multiple local sources exist", () => {
    render(
      <EnvironmentSection
        environment={summary({
          contractSources: [{ file: ".env.example", variableCount: 1 }],
          localSources: [
            { file: ".env.local", variableCount: 1 },
            { file: ".env", variableCount: 1 },
          ],
          variables: [
            variable({
              name: "DATABASE_URL",
              declaredIn: [".env.example"],
              satisfied: true,
              source: ".env.local",
            }),
          ],
        })}
      />,
    );

    expect(screen.getByText(".env.local")).toBeInTheDocument();
    expect(screen.getByTitle(".env.local · .env")).toBeInTheDocument();
  });

  it("shows a concise 'process' label when satisfied via the process environment among multiple sources", () => {
    render(
      <EnvironmentSection
        environment={summary({
          contractSources: [{ file: ".env.example", variableCount: 1 }],
          localSources: [{ file: ".env", variableCount: 1 }],
          variables: [
            variable({
              name: "HOME",
              declaredIn: [".env.example"],
              satisfied: true,
              source: "process environment",
            }),
          ],
        })}
      />,
    );

    expect(screen.getByText("process")).toBeInTheDocument();
  });

  it("shows multiple Environment source names in the column headers", () => {
    render(
      <EnvironmentSection
        environment={summary({
          contractSources: [
            { file: ".env.example", variableCount: 1 },
            { file: ".env.sample", variableCount: 1 },
          ],
          localSources: [
            { file: ".env.local", variableCount: 1 },
            { file: ".env", variableCount: 1 },
          ],
          variables: [variable({ name: "A", declaredIn: [".env.example", ".env.sample"] })],
        })}
      />,
    );

    expect(screen.getByTitle(".env.local · .env")).toBeInTheDocument();
    expect(screen.getByTitle(".env.example · .env.sample")).toBeInTheDocument();
  });

  it("shows a clear empty state when no contract is declared and no usage was found", () => {
    render(<EnvironmentSection environment={summary({ hasContract: false, variables: [] })} />);

    expect(screen.getByText(/no environment contract declared/i)).toBeInTheDocument();
  });

  it("surfaces an analysis error without rendering variable data", () => {
    render(<EnvironmentSection environment={summary({ error: "permission denied" })} />);

    expect(screen.getByText(/environment analysis failed/i)).toBeInTheDocument();
    expect(screen.getByText(/permission denied/i)).toBeInTheDocument();
  });
});
