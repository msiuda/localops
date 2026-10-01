import { ArrowLeft } from "lucide-react";
import { Link, useNavigate, useSearch } from "@tanstack/react-router";
import { HealthBadge, useProjectDetailQuery } from "@/entities/project";
import { Badge } from "@/shared/ui/atoms/Badge";
import { MonoText } from "@/shared/ui/atoms/MonoText";
import { formatProjectPath } from "@/shared/lib/formatProjectPath";
import { PROJECT_DETAIL_TABS, PROJECT_DETAIL_TAB_LABELS } from "./model/tabs";
import { OverviewSection } from "./sections/OverviewSection";
import { DoctorSection } from "./sections/DoctorSection";
import { EnvironmentSection } from "./sections/EnvironmentSection";
import { ValidationSection } from "./sections/ValidationSection";

/** The route component for "/project" (registered as such in routes/project.tsx). */
export function ProjectDetailPage() {
  const { path, tab } = useSearch({ from: "/project" });
  const { data: detail, isPending, isError, error } = useProjectDetailQuery(path);
  const navigate = useNavigate({ from: "/project" });

  const isUnavailable = detail?.health === "unavailable";

  return (
    <div className="flex h-full min-w-0 flex-col">
      <header className="drag-region border-border flex items-start gap-5 border-b px-7 pt-4.5 pb-4">
        <Link
          to="/"
          aria-label="Back to Projects"
          className="no-drag text-text-secondary hover:bg-surface-hover hover:text-text-primary focus-visible:outline-accent flex flex-shrink-0 cursor-pointer items-center gap-1.5 rounded-sm py-1.5 pr-2.5 pl-1.5 text-[12.5px] font-semibold focus-visible:outline-2 focus-visible:outline-offset-1"
        >
          <ArrowLeft size={16} strokeWidth={2.2} />
          Projects
        </Link>

        {detail && (
          <div className="flex min-w-0 flex-1 flex-col gap-1.5">
            <div className="flex items-center gap-2.5">
              <h1 className="text-text-primary m-0 text-[17px] font-bold tracking-tight">{detail.name}</h1>
              <HealthBadge health={detail.health} />
            </div>
            <MonoText className="text-text-muted truncate text-[11.5px]" title={detail.path}>
              {formatProjectPath(detail.path)}
            </MonoText>
            {!isUnavailable && (
              <div className="mt-0.5 flex flex-wrap gap-1.5">
                {detail.technologies?.map((tech) => (
                  <Badge key={tech}>{tech}</Badge>
                ))}
                {detail.packageManager ? <Badge>{detail.packageManager}</Badge> : null}
              </div>
            )}
          </div>
        )}
      </header>

      {isPending && <div className="text-text-muted px-7 py-7 text-[12.5px]">Loading…</div>}

      {isError && (
        <div className="text-danger px-7 py-7 text-[12.5px]">
          Couldn't load this project: {error instanceof Error ? error.message : String(error)}
        </div>
      )}

      {detail && isUnavailable && (
        <div className="text-text-muted px-7 py-7 text-[12.5px]">
          {detail.unavailableReason || "Project path is unavailable"}
        </div>
      )}

      {detail && !isUnavailable && (
        <>
          <div
            className="border-border flex gap-5 border-b px-7"
            role="tablist"
            aria-label="Project detail sections"
          >
            {PROJECT_DETAIL_TABS.map((key) => (
              <button
                key={key}
                type="button"
                role="tab"
                aria-selected={tab === key}
                className={`hover:text-text-primary focus-visible:outline-accent relative py-2.5 text-[12.5px] font-semibold after:absolute after:right-0 after:-bottom-px after:left-0 after:h-0.5 after:rounded-sm after:content-[''] focus-visible:outline-2 focus-visible:outline-offset-2 ${
                  tab === key ? "text-text-primary after:bg-accent" : "text-text-muted after:bg-transparent"
                }`}
                onClick={() => navigate({ search: (prev) => ({ ...prev, tab: key }) })}
              >
                {PROJECT_DETAIL_TAB_LABELS[key]}
              </button>
            ))}
          </div>

          <div className="flex-1 overflow-y-auto px-7 py-5">
            {tab === "overview" && <OverviewSection detail={detail} />}
            {tab === "doctor" && <DoctorSection checks={detail.doctorChecks ?? []} />}
            {tab === "environment" && <EnvironmentSection environment={detail.environment} />}
            {tab === "validation" && <ValidationSection />}
          </div>
        </>
      )}
    </div>
  );
}
