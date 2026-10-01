import { Service } from "@bindings/github.com/msiuda/localops/internal/desktop";

/**
 * The Project entity's Wails API boundary: the ONLY place that calls the
 * generated Service binding directly. Query/mutation hooks (model/queries.ts)
 * go through this module instead of importing the binding themselves, so
 * tests can mock this small surface without a real Wails process. Types are
 * re-exported from the generated bindings, never redefined.
 */
export const projectApi = {
  getOverview: () => Service.GetOverview(),
  getProjectDetail: (path: string) => Service.GetProjectDetail(path),
  addProject: (path: string) => Service.AddProject(path),
  pickProjectDirectory: () => Service.PickProjectDirectory(),
};
