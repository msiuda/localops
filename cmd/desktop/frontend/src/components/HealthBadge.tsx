import { CheckCircle2, AlertTriangle, HelpCircle } from "lucide-react";
import { Health } from "../../bindings/github.com/msiuda/localops/internal/desktop";
import "./HealthBadge.css";

const CONFIG: Record<string, { label: string; icon: typeof CheckCircle2; className: string }> = {
  [Health.HealthHealthy]: { label: "Healthy", icon: CheckCircle2, className: "health-badge--healthy" },
  [Health.HealthIssues]: { label: "Issues", icon: AlertTriangle, className: "health-badge--issues" },
  [Health.HealthUnavailable]: { label: "Unavailable", icon: HelpCircle, className: "health-badge--unavailable" },
};

interface HealthBadgeProps {
  health: string;
}

export function HealthBadge({ health }: HealthBadgeProps) {
  const config = CONFIG[health] ?? CONFIG[Health.HealthUnavailable];
  const Icon = config.icon;

  return (
    <span className={`health-badge ${config.className}`}>
      <Icon size={12} strokeWidth={2.5} />
      {config.label}
    </span>
  );
}
