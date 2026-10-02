import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { OverviewSection } from "./OverviewSection";
import { Health, type ProjectDetail } from "@/entities/project";

function detail(overrides: Partial<ProjectDetail>): ProjectDetail {
  return {
    name: "demo",
    path: "/tmp/demo",
    health: Health.HealthHealthy,
    unavailableReason: "",
    technologies: [],
    moreTechnologies: [],
    technologyGroups: [],
    packageManager: "",
    issueCount: 0,
    doctorChecks: [],
    environment: {
      hasContract: false,
      contractSources: [],
      localSources: [],
      variables: [],
      findings: [],
      missingCount: 0,
      undeclaredCount: 0,
      error: "",
    },
    ...overrides,
  };
}

describe("OverviewSection", () => {
  it("shows the grouped Stack picture by kind", () => {
    render(
      <OverviewSection
        detail={detail({
          technologyGroups: [
            { label: "Languages", names: ["TypeScript"] },
            { label: "Runtime", names: ["Node.js"] },
            { label: "Frameworks", names: ["NestJS"] },
          ],
        })}
      />,
    );

    expect(screen.getByText("Languages")).toBeInTheDocument();
    expect(screen.getByText("TypeScript")).toBeInTheDocument();
    expect(screen.getByText("Runtime")).toBeInTheDocument();
    expect(screen.getByText("Node.js")).toBeInTheDocument();
    expect(screen.getByText("Frameworks")).toBeInTheDocument();
    expect(screen.getByText("NestJS")).toBeInTheDocument();
  });

  it("collapses many frameworks into a restrained +N overflow badge", () => {
    render(
      <OverviewSection
        detail={detail({
          technologyGroups: [
            { label: "Frameworks", names: ["React", "Redux", "React Router", "Zustand", "Vite"] },
          ],
        })}
      />,
    );

    expect(screen.getByText("React")).toBeInTheDocument();
    expect(screen.getByText("Zustand")).toBeInTheDocument();
    expect(screen.getByText("+1")).toBeInTheDocument();
    expect(screen.queryByText("Vite")).not.toBeInTheDocument();
  });

  it("shows a calm empty state when nothing was detected", () => {
    render(<OverviewSection detail={detail({ technologyGroups: [] })} />);

    expect(screen.getByText("None detected")).toBeInTheDocument();
  });
});
