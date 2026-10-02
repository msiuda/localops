import { describe, expect, it, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderApp } from "@/shared/lib/test/renderApp";
import { projectApi } from "@/entities/project/api/projectApi";
import { Health, type ProjectCard } from "@/entities/project";

vi.mock("@/entities/project/api/projectApi", () => ({
  projectApi: {
    getOverview: vi.fn(),
    getProjectDetail: vi.fn(),
    addProject: vi.fn(),
    pickProjectDirectory: vi.fn(),
  },
}));

const mockedApi = vi.mocked(projectApi);

function card(overrides: Partial<ProjectCard>): ProjectCard {
  return {
    name: "demo",
    path: "/tmp/demo",
    health: Health.HealthHealthy,
    technologies: [],
    moreTechnologies: [],
    packageManager: "",
    issueCount: 0,
    findings: [],
    unavailableReason: "",
    ...overrides,
  };
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe("ProjectsPage", () => {
  it("renders project rows from query data with health presentation", async () => {
    mockedApi.getOverview.mockResolvedValue({
      projects: [
        card({ name: "healthy-app", path: "/tmp/healthy-app", health: Health.HealthHealthy }),
        card({
          name: "issues-app",
          path: "/tmp/issues-app",
          health: Health.HealthIssues,
          issueCount: 1,
          findings: [{ tool: "go", detail: "outdated" }],
        }),
        card({
          name: "gone-app",
          path: "/tmp/gone-app",
          health: Health.HealthUnavailable,
          unavailableReason: "path does not exist",
        }),
      ],
    });

    renderApp("/");

    expect(await screen.findByText("healthy-app")).toBeInTheDocument();
    expect(screen.getByText("issues-app")).toBeInTheDocument();
    expect(screen.getByText("gone-app")).toBeInTheDocument();

    expect(screen.getByText("Healthy")).toBeInTheDocument();
    expect(screen.getByText("Issues")).toBeInTheDocument();
    expect(screen.getByText("Unavailable")).toBeInTheDocument();
    expect(screen.getByText("1 issue")).toBeInTheDocument();
    expect(screen.getByText("path does not exist")).toBeInTheDocument();
  });

  it("shows the compact technology summary with a +N overflow, never Git", async () => {
    mockedApi.getOverview.mockResolvedValue({
      projects: [
        card({
          name: "billsy-api",
          path: "/tmp/billsy-api",
          technologies: ["TypeScript", "NestJS", "Node.js"],
          moreTechnologies: ["Express"],
          packageManager: "yarn",
        }),
      ],
    });

    renderApp("/");

    expect(await screen.findByText("billsy-api")).toBeInTheDocument();
    expect(screen.getByText("TypeScript")).toBeInTheDocument();
    expect(screen.getByText("NestJS")).toBeInTheDocument();
    expect(screen.getByText("Node.js")).toBeInTheDocument();
    expect(screen.getByText(/yarn/)).toBeInTheDocument();
    expect(screen.getByText("+1")).toBeInTheDocument();
    expect(screen.queryByText("Git")).not.toBeInTheDocument();
  });

  it("shows the empty state with an Add Project action when there are no projects", async () => {
    mockedApi.getOverview.mockResolvedValue({ projects: [] });

    renderApp("/");

    expect(await screen.findByText("No projects yet")).toBeInTheDocument();
    // Two Add Project buttons: header + empty state.
    expect(screen.getAllByRole("button", { name: /add project/i })).toHaveLength(2);
  });

  it("navigates to Project Detail when a project row is clicked", async () => {
    mockedApi.getOverview.mockResolvedValue({
      projects: [card({ name: "healthy-app", path: "/tmp/healthy-app", health: Health.HealthHealthy })],
    });
    mockedApi.getProjectDetail.mockResolvedValue({
      name: "healthy-app",
      path: "/tmp/healthy-app",
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
    });

    renderApp("/");
    const user = userEvent.setup();

    const row = await screen.findByText("healthy-app");
    await user.click(row);

    expect(await screen.findByRole("link", { name: "Back to Projects" })).toBeInTheDocument();
    await waitFor(() => expect(mockedApi.getProjectDetail).toHaveBeenCalledWith("/tmp/healthy-app"));
  });

  it("adds a project and reloads the overview on success", async () => {
    mockedApi.getOverview.mockResolvedValueOnce({ projects: [] }).mockResolvedValueOnce({
      projects: [card({ name: "new-project", path: "/tmp/new-project", health: Health.HealthHealthy })],
    });
    mockedApi.pickProjectDirectory.mockResolvedValue("/tmp/new-project");
    mockedApi.addProject.mockResolvedValue(undefined);

    renderApp("/");
    const user = userEvent.setup();

    await screen.findByText("No projects yet");
    const [addButton] = screen.getAllByRole("button", { name: /add project/i });
    await user.click(addButton);

    expect(await screen.findByText("new-project")).toBeInTheDocument();
    expect(mockedApi.addProject).toHaveBeenCalledWith("/tmp/new-project");
  });

  it("does nothing when the directory picker is cancelled", async () => {
    mockedApi.getOverview.mockResolvedValue({ projects: [] });
    mockedApi.pickProjectDirectory.mockResolvedValue("");

    renderApp("/");
    const user = userEvent.setup();

    const [addButton] = await screen.findAllByRole("button", { name: /add project/i });
    await user.click(addButton);

    await waitFor(() => expect(mockedApi.pickProjectDirectory).toHaveBeenCalled());
    expect(mockedApi.addProject).not.toHaveBeenCalled();
  });

  it("shows an inline error when adding a project fails", async () => {
    mockedApi.getOverview.mockResolvedValue({ projects: [] });
    mockedApi.pickProjectDirectory.mockResolvedValue("/tmp/broken");
    mockedApi.addProject.mockRejectedValue(new Error("project already registered"));

    renderApp("/");
    const user = userEvent.setup();

    const [addButton] = await screen.findAllByRole("button", { name: /add project/i });
    await user.click(addButton);

    expect(await screen.findByText(/couldn't add project/i)).toBeInTheDocument();
    expect(screen.getByText(/project already registered/i)).toBeInTheDocument();
  });
});
