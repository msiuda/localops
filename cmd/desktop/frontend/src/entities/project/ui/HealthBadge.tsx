import { CheckCircle2, AlertTriangle, HelpCircle, type LucideIcon } from "lucide-react";
import { StatusBadge, type StatusTone } from "@/shared/ui/atoms/StatusBadge";
import { Health } from "../model/types";

const CONFIG: Record<string, { label: string; icon: LucideIcon; tone: StatusTone }> = {
  [Health.HealthHealthy]: { label: "Healthy", icon: CheckCircle2, tone: "success" },
  [Health.HealthIssues]: { label: "Issues", icon: AlertTriangle, tone: "warning" },
  [Health.HealthUnavailable]: { label: "Unavailable", icon: HelpCircle, tone: "muted" },
};

/** Maps a project's Health to the generic StatusBadge — entity-specific presentation. */
export function HealthBadge({ health }: { health: string }) {
  const config = CONFIG[health] ?? CONFIG[Health.HealthUnavailable];
  return <StatusBadge label={config.label} tone={config.tone} icon={config.icon} />;
}
