import type { ReactNode } from "react";

interface EmptyStateProps {
  icon: ReactNode;
  title: string;
  description: string;
  action?: ReactNode;
}

export function EmptyState({ icon, title, description, action }: EmptyStateProps) {
  return (
    <div className="border-border-strong text-text-secondary flex flex-col items-center justify-center gap-1.5 rounded-md border border-dashed px-6 py-16 text-center">
      <div className="text-text-muted mb-2.5">{icon}</div>
      <p className="text-text-primary text-emphasis m-0 font-semibold">{title}</p>
      <p className="text-text-muted text-body m-0 max-w-80">{description}</p>
      {action ? <div className="mt-1.5">{action}</div> : null}
    </div>
  );
}
