import { FolderGit2, AlertCircle } from "lucide-react";
import { ProjectRow, ProjectRowSkeleton, useProjectsOverviewQuery } from "@/entities/project";
import { EmptyState } from "@/shared/ui/molecules/EmptyState";
import { AddProjectButton } from "@/features/add-project/ui/AddProjectButton";

export function ProjectsPage() {
  const { data, isPending, isError, error } = useProjectsOverviewQuery();
  const projects = data?.projects ?? [];

  return (
    <div className="flex h-full min-w-0 flex-col">
      <header className="drag-region flex items-start justify-between px-7 pt-5 pb-4">
        <div>
          <h1 className="text-text-primary text-title m-0 font-bold tracking-tight">Projects</h1>
          <p className="text-text-muted text-body mt-1 mb-0">Your local development workspace</p>
        </div>
        <AddProjectButton className="no-drag" />
      </header>

      <div className="flex-1 overflow-y-auto px-7 pb-7">
        {isPending && (
          <div className="flex flex-col gap-2">
            <ProjectRowSkeleton />
            <ProjectRowSkeleton />
            <ProjectRowSkeleton />
          </div>
        )}

        {isError && (
          <EmptyState
            icon={<AlertCircle size={28} strokeWidth={1.5} />}
            title="Couldn't load projects"
            description={error instanceof Error ? error.message : String(error)}
          />
        )}

        {!isPending && !isError && projects.length === 0 && (
          <EmptyState
            icon={<FolderGit2 size={28} strokeWidth={1.5} />}
            title="No projects yet"
            description="LocalOps tracks local projects you register. Add one to see its health here."
            action={<AddProjectButton />}
          />
        )}

        {!isPending && !isError && projects.length > 0 && (
          <div className="flex flex-col gap-2">
            {projects.map((project) => (
              <ProjectRow key={project.path} project={project} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
