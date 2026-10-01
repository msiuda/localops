import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { EnvironmentSection } from "./EnvironmentSection";
import type { EnvironmentSummary } from "@/entities/project";

function summary(overrides: Partial<EnvironmentSummary>): EnvironmentSummary {
  return {
    hasContract: true,
    contractSources: [],
    localSources: [],
    variables: [],
    findings: [],
    missingCount: 0,
    error: "",
    ...overrides,
  };
}

describe("EnvironmentSection", () => {
  it("shows variable names and satisfied/missing status without any value", () => {
    render(
      <EnvironmentSection
        environment={summary({
          contractSources: [{ file: ".env.example", variableCount: 2 }],
          variables: [
            { name: "DATABASE_URL", declaredIn: [".env.example"], satisfied: true, source: ".env.local" },
            { name: "API_KEY", declaredIn: [".env.example"], satisfied: false, source: "" },
          ],
        })}
      />,
    );

    expect(screen.getByText("DATABASE_URL")).toBeInTheDocument();
    expect(screen.getByText("API_KEY")).toBeInTheDocument();
    expect(screen.getByText("Satisfied")).toBeInTheDocument();
    expect(screen.getByText("· .env.local")).toBeInTheDocument();
    expect(screen.getByText("Missing")).toBeInTheDocument();

    // The rendered output must never contain a secret-shaped value, only
    // names/status/sources — this is a coarse guard, not a substitute for
    // the backend's own structural guarantee that EnvironmentSummary has
    // no field capable of holding a value.
    expect(document.body.textContent).not.toMatch(/postgres:\/\//);
  });

  it("shows a clear empty state when no contract is declared", () => {
    render(<EnvironmentSection environment={summary({ hasContract: false })} />);

    expect(screen.getByText(/no environment contract declared/i)).toBeInTheDocument();
  });

  it("surfaces an analysis error without rendering variable data", () => {
    render(<EnvironmentSection environment={summary({ error: "permission denied" })} />);

    expect(screen.getByText(/environment analysis failed/i)).toBeInTheDocument();
    expect(screen.getByText(/permission denied/i)).toBeInTheDocument();
  });
});
