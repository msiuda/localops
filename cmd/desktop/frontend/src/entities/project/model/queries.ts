import { useQuery, useQueryClient, type UseQueryResult } from "@tanstack/react-query";
import { projectApi } from "../api/projectApi";
import type { ProjectDetail, ProjectsOverview } from "./types";

/**
 * Centralized query keys for the Project entity. Every hook below builds
 * its key from here — components/features should never write a raw query
 * key array themselves.
 */
export const projectKeys = {
  all: ["projects"] as const,
  overview: () => [...projectKeys.all, "overview"] as const,
  detail: (path: string) => [...projectKeys.all, "detail", path] as const,
};

export function useProjectsOverviewQuery(): UseQueryResult<ProjectsOverview> {
  return useQuery({
    queryKey: projectKeys.overview(),
    queryFn: projectApi.getOverview,
  });
}

export function useProjectDetailQuery(path: string): UseQueryResult<ProjectDetail> {
  return useQuery({
    queryKey: projectKeys.detail(path),
    queryFn: () => projectApi.getProjectDetail(path),
    enabled: path.length > 0,
  });
}

/** Invalidates the Projects overview query — call after a mutation that changes the registry. */
export function useInvalidateProjectsOverview() {
  const queryClient = useQueryClient();
  return () => queryClient.invalidateQueries({ queryKey: projectKeys.overview() });
}
