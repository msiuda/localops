import { ChevronRight } from "lucide-react";
import { Link } from "@tanstack/react-router";
import { Badge } from "@/shared/ui/atoms/Badge";
import { MonoText } from "@/shared/ui/atoms/MonoText";
import { formatProjectPath } from "@/shared/lib/formatProjectPath";
import { HealthBadge } from "./HealthBadge";
import type { ProjectCard } from "../model/types";

interface ProjectRowProps {
  project: ProjectCard;
}

/**
 * Explicit 4-column grid: identity | stack | issues | chevron. Each row is
 * its own grid container (a <Link>), so the stack/issue/chevron columns
 * use FIXED widths rather than minmax() ranges — a minmax range can resolve
 * to a different pixel width per row depending on that row's own content,
 * which would silently break the cross-row alignment this layout is meant
 * to guarantee. Only the identity column (1fr) is flexible.
 */
export function ProjectRow({ project }: ProjectRowProps) {
  const isUnavailable = project.health === "unavailable";
  const previewFinding = project.findings?.[0];

  return (
    <Link
      to="/project"
      search={{ path: project.path, tab: "overview" }}
      className="border-border bg-surface hover:border-border-strong hover:bg-surface-hover focus-visible:outline-accent grid cursor-pointer items-center gap-4 rounded-md border px-4 py-3 text-left transition-colors duration-100 focus-visible:outline-2 focus-visible:-outline-offset-1"
      style={{ gridTemplateColumns: "minmax(180px,1fr) 180px 150px 20px" }}
    >
      <div className="flex min-w-0 flex-col gap-1">
        <div className="flex items-center gap-2.5">
          <span className="text-text-primary text-emphasis truncate font-semibold">{project.name}</span>
          <HealthBadge health={project.health} />
        </div>
        <MonoText className="text-text-muted text-micro truncate" title={project.path}>
          {formatProjectPath(project.path)}
        </MonoText>
      </div>

      {isUnavailable ? (
        <span className="text-text-muted text-micro col-span-2 truncate">
          {project.unavailableReason || "Project path is unavailable"}
        </span>
      ) : (
        <>
          <div className="flex min-w-0 flex-wrap gap-1.5">
            {project.technologies?.map((tech) => (
              <Badge key={tech}>{tech}</Badge>
            ))}
            {project.packageManager ? <Badge>{project.packageManager}</Badge> : null}
          </div>

          <div className="flex min-w-0 flex-col items-end gap-0.5">
            {project.issueCount > 0 ? (
              <>
                <span className="text-warning text-micro font-semibold whitespace-nowrap">
                  {project.issueCount} issue{project.issueCount === 1 ? "" : "s"}
                </span>
                {previewFinding ? (
                  <span className="text-text-muted text-micro max-w-full truncate">
                    {previewFinding.tool} — {previewFinding.detail}
                  </span>
                ) : null}
              </>
            ) : (
              <span className="text-text-muted text-micro whitespace-nowrap">No issues</span>
            )}
          </div>
        </>
      )}

      <ChevronRight className="text-text-muted flex-shrink-0" size={16} strokeWidth={2} aria-hidden="true" />
    </Link>
  );
}
