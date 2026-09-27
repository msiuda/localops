import { useCallback, useEffect, useState } from "react";
import { Service, ProjectCard } from "../../../bindings/github.com/msiuda/localops/internal/desktop";

export type OverviewState =
  | { status: "loading" }
  | { status: "empty" }
  | { status: "loaded"; projects: ProjectCard[] }
  | { status: "error"; message: string };

/**
 * Loads the registered-project Overview through the generated Wails
 * binding (Service.GetOverview), which in turn runs the existing
 * storage → project inspection → Doctor → Overview chain in Go. This hook
 * only tracks loading/empty/loaded/error state; it never recomputes health.
 */
export function useOverview() {
  const [state, setState] = useState<OverviewState>({ status: "loading" });

  const refresh = useCallback(() => {
    setState({ status: "loading" });
    Service.GetOverview()
      .then((result) => {
        const projects = result.projects ?? [];
        setState(projects.length === 0 ? { status: "empty" } : { status: "loaded", projects });
      })
      .catch((err: unknown) => {
        setState({
          status: "error",
          message: err instanceof Error ? err.message : String(err),
        });
      });
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  return { state, refresh };
}
