import { ProjectCard } from "../../bindings/github.com/msiuda/localops/internal/desktop";
import { HealthBadge } from "./HealthBadge";
import { TechPill } from "./TechPill";
import "./ProjectRow.css";

interface ProjectRowProps {
  project: ProjectCard;
}

/** Replaces a leading /Users/<name> or /home/<name> with ~ for readability. */
function shortenPath(path: string): string {
  return path.replace(/^\/(Users|home)\/[^/]+/, "~");
}

export function ProjectRow({ project }: ProjectRowProps) {
  const isUnavailable = project.health === "unavailable";
  const previewFinding = project.findings?.[0];

  return (
    <div className="project-row">
      <div className="project-row__main">
        <div className="project-row__heading">
          <span className="project-row__name">{project.name}</span>
          <HealthBadge health={project.health} />
        </div>
        <span className="project-row__path">{shortenPath(project.path)}</span>
      </div>

      {isUnavailable ? (
        <div className="project-row__side">
          <span className="project-row__unavailable">
            {project.unavailableReason || "Project path is unavailable"}
          </span>
        </div>
      ) : (
        <div className="project-row__side">
          <div className="project-row__pills">
            {project.technologies?.map((tech) => (
              <TechPill key={tech} label={tech} />
            ))}
            {project.packageManager ? <TechPill label={project.packageManager} /> : null}
          </div>

          <div className="project-row__issues">
            {project.issueCount > 0 ? (
              <>
                <span className="project-row__issue-count">
                  {project.issueCount} issue{project.issueCount === 1 ? "" : "s"}
                </span>
                {previewFinding ? (
                  <span className="project-row__issue-preview">
                    {previewFinding.tool} — {previewFinding.detail}
                  </span>
                ) : null}
              </>
            ) : (
              <span className="project-row__ok">No issues</span>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
