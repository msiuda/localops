import { describe, expect, it, vi, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderApp } from "@/shared/lib/test/renderApp";
import { projectApi } from "@/entities/project/api/projectApi";
import { Health, type ProjectDetail } from "@/entities/project";

vi.mock("@/entities/project/api/projectApi", () => ({
  projectApi: {
    getOverview: vi.fn(),
    getProjectDetail: vi.fn(),
    addProject: vi.fn(),
    pickProjectDirectory: vi.fn(),
  },
}));

const mockedApi = vi.mocked(projectApi);

function detail(overrides: Partial<ProjectDetail>): ProjectDetail {
  return {
    name: "demo",
    path: "/tmp/demo",
    health: Health.HealthHealthy,
    unavailableReason: "",
    technologies: ["Go"],
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
      error: "",
    },
    ...overrides,
  };
}

beforeEach(() => {
  vi.clearAllMocks();
  mockedApi.getOverview.mockResolvedValue({ projects: [] });
});

describe("ProjectDetailPage", () => {
  it("displays the selected project from the route's search params", async () => {
    mockedApi.getProjectDetail.mockResolvedValue(detail({ name: "demo", path: "/tmp/demo" }));

    renderApp("/project?path=%2Ftmp%2Fdemo&tab=overview");

    expect(await screen.findByRole("heading", { name: "demo" })).toBeInTheDocument();
    expect(screen.getByText("/tmp/demo")).toBeInTheDocument();
    expect(mockedApi.getProjectDetail).toHaveBeenCalledWith("/tmp/demo");
  });

  it("returns to Projects via the Back action", async () => {
    mockedApi.getProjectDetail.mockResolvedValue(detail({}));

    renderApp("/project?path=%2Ftmp%2Fdemo&tab=overview");
    const user = userEvent.setup();

    await screen.findByRole("heading", { name: "demo" });
    await user.click(screen.getByRole("link", { name: "Back to Projects" }));

    expect(await screen.findByText("Your local development workspace")).toBeInTheDocument();
  });

  it("switches between Doctor, Environment, and Validation sections", async () => {
    mockedApi.getProjectDetail.mockResolvedValue(
      detail({
        doctorChecks: [{ tool: "git", available: true, version: "2.40.0", ok: true, note: "", detail: "" }],
        environment: {
          hasContract: true,
          contractSources: [{ file: ".env.example", variableCount: 1 }],
          localSources: [],
          variables: [{ name: "API_KEY", declaredIn: [".env.example"], satisfied: false, source: "" }],
          findings: [],
          missingCount: 1,
          error: "",
        },
      }),
    );

    renderApp("/project?path=%2Ftmp%2Fdemo&tab=overview");
    const user = userEvent.setup();
    await screen.findByRole("heading", { name: "demo" });

    await user.click(screen.getByRole("tab", { name: "Doctor" }));
    expect(await screen.findByText("git")).toBeInTheDocument();
    expect(screen.getByText("Passed")).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Environment" }));
    expect(await screen.findByText("API_KEY")).toBeInTheDocument();
    expect(screen.getByText("Missing")).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Validation" }));
    expect(await screen.findByText(/isn't runnable from the desktop yet/i)).toBeInTheDocument();
  });
});
