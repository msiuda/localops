import { ArrowLeft } from "lucide-react";
import { Link, useNavigate, useSearch } from "@tanstack/react-router";
import { CompactTechStack, HealthBadge, useProjectDetailQuery } from "@/entities/project";
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
      <header className="drag-region border-border flex items-start gap-5 border-b px-7 pt-5 pb-4">
        <Link
          to="/"
          aria-label="Back to Projects"
          className="no-drag text-text-secondary hover:bg-surface-hover hover:text-text-primary focus-visible:outline-accent text-emphasis flex flex-shrink-0 cursor-pointer items-center gap-1.5 rounded-sm py-1.5 pr-2.5 pl-1.5 font-semibold focus-visible:outline-2 focus-visible:outline-offset-1"
        >
          <ArrowLeft size={16} strokeWidth={2.2} />
          Projects
        </Link>

        {detail && (
          <div className="flex min-w-0 flex-1 flex-col gap-1.5">
            <div className="flex items-center gap-2.5">
              <h1 className="text-text-primary text-title-sm m-0 font-bold tracking-tight">{detail.name}</h1>
              <HealthBadge health={detail.health} />
            </div>
            <MonoText className="text-text-muted text-micro truncate" title={detail.path}>
              {formatProjectPath(detail.path)}
            </MonoText>
            {!isUnavailable && (
              <CompactTechStack
                className="mt-0.5"
                technologies={detail.technologies}
                moreTechnologies={detail.moreTechnologies}
                packageManager={detail.packageManager}
              />
            )}
          </div>
        )}
      </header>

      {isPending && <div className="text-text-muted text-body px-7 py-7">Loading…</div>}

      {isError && (
        <div className="text-danger text-body px-7 py-7">
          Couldn't load this project: {error instanceof Error ? error.message : String(error)}
        </div>
      )}

      {detail && isUnavailable && (
        <div className="text-text-muted text-body px-7 py-7">
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
                className={`hover:bg-surface-hover hover:text-text-primary focus-visible:outline-accent text-emphasis relative -mx-1.5 rounded-t-sm px-1.5 py-2.5 font-semibold after:absolute after:right-1.5 after:-bottom-px after:left-1.5 after:h-0.5 after:rounded-sm after:content-[''] focus-visible:outline-2 focus-visible:outline-offset-2 ${
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
