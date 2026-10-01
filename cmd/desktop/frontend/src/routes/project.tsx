import { createFileRoute } from "@tanstack/react-router";
import { ProjectDetailPage } from "@/features/project-detail/ProjectDetailPage";
import type { ProjectDetailTab } from "@/features/project-detail/model/tabs";
import { PROJECT_DETAIL_TABS } from "@/features/project-detail/model/tabs";

/**
 * Project identity is an absolute filesystem path, which must never become
 * a raw URL path segment (illegal characters, no pretty encoding, and it
 * would leak local filesystem layout into what looks like a routable URL).
 * Instead this route lives at a fixed path ("/project") and carries the
 * path as a typed search parameter, validated through TanStack Router's
 * own validateSearch — no Zod needed for a shape this small.
 */
interface ProjectSearch {
  path: string;
  tab: ProjectDetailTab;
}

export const Route = createFileRoute("/project")({
  validateSearch: (search: Record<string, unknown>): ProjectSearch => ({
    path: typeof search.path === "string" ? search.path : "",
    tab: PROJECT_DETAIL_TABS.includes(search.tab as ProjectDetailTab)
      ? (search.tab as ProjectDetailTab)
      : "overview",
  }),
  component: ProjectDetailPage,
});
