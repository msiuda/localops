export const PROJECT_DETAIL_TABS = ["overview", "doctor", "environment", "validation"] as const;

export type ProjectDetailTab = (typeof PROJECT_DETAIL_TABS)[number];

export const PROJECT_DETAIL_TAB_LABELS: Record<ProjectDetailTab, string> = {
  overview: "Overview",
  doctor: "Doctor",
  environment: "Environment",
  validation: "Validation",
};
