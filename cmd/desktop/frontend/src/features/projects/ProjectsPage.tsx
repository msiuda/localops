import { FolderGit2, RefreshCw, AlertCircle } from "lucide-react";
import { ProjectRow } from "../../components/ProjectRow";
import { SkeletonRow } from "../../components/SkeletonRow";
import { EmptyState } from "../../components/EmptyState";
import { useOverview } from "./useOverview";
import "./ProjectsPage.css";

export function ProjectsPage() {
  const { state, refresh } = useOverview();

  return (
    <div className="projects-page">
      <header className="projects-page__header drag-region">
        <div>
          <h1 className="projects-page__title">Projects</h1>
          <p className="projects-page__subtitle">Your local development workspace</p>
        </div>
        <button
          type="button"
          className="projects-page__refresh no-drag"
          onClick={refresh}
          disabled={state.status === "loading"}
          aria-label="Refresh projects"
        >
          <RefreshCw size={14} strokeWidth={2.2} className={state.status === "loading" ? "spin" : undefined} />
          Refresh
        </button>
      </header>

      <div className="projects-page__content">
        {state.status === "loading" && (
          <div className="projects-page__list">
            <SkeletonRow />
            <SkeletonRow />
            <SkeletonRow />
          </div>
        )}

        {state.status === "error" && (
          <EmptyState
            icon={<AlertCircle size={28} strokeWidth={1.5} />}
            title="Couldn't load projects"
            description={state.message}
          />
        )}

        {state.status === "empty" && (
          <EmptyState
            icon={<FolderGit2 size={28} strokeWidth={1.5} />}
            title="No projects yet"
            description="Register a project from the CLI to see it here: localops project add <path>"
          />
        )}

        {state.status === "loaded" && (
          <div className="projects-page__list">
            {state.projects.map((project) => (
              <ProjectRow key={project.path} project={project} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
