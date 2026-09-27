import { Boxes, Stethoscope, ShieldCheck, KeyRound, Settings, type LucideIcon } from "lucide-react";
import { WindowControls } from "./WindowControls";
import "./Sidebar.css";

export type NavKey = "projects" | "doctor" | "validation" | "environment";

interface NavItemConfig {
  key: NavKey;
  label: string;
  icon: LucideIcon;
  enabled: boolean;
}

const NAV_ITEMS: NavItemConfig[] = [
  { key: "projects", label: "Projects", icon: Boxes, enabled: true },
  { key: "doctor", label: "Doctor", icon: Stethoscope, enabled: false },
  { key: "validation", label: "Validation", icon: ShieldCheck, enabled: false },
  { key: "environment", label: "Environment", icon: KeyRound, enabled: false },
];

interface SidebarProps {
  active: NavKey;
}

/**
 * The application sidebar: window controls, wordmark, primary navigation,
 * and a pinned Settings entry. Doctor / Validation / Environment are
 * intentionally non-functional in this milestone — they exist to establish
 * LocalOps's information architecture, not to be usable yet.
 */
export function Sidebar({ active }: SidebarProps) {
  return (
    <aside className="sidebar">
      <div className="sidebar__top drag-region">
        <WindowControls />
        <div className="sidebar__brand">
          <span className="sidebar__brand-mark">L</span>
          <span className="sidebar__brand-name">LocalOps</span>
        </div>
      </div>

      <nav className="sidebar__nav">
        {NAV_ITEMS.map((item) => {
          const Icon = item.icon;
          const isActive = item.key === active;
          return (
            <button
              key={item.key}
              type="button"
              className={`sidebar__nav-item${isActive ? " sidebar__nav-item--active" : ""}`}
              disabled={!item.enabled}
              title={item.enabled ? undefined : "Coming soon"}
            >
              <Icon size={16} strokeWidth={2} />
              <span>{item.label}</span>
            </button>
          );
        })}
      </nav>

      <div className="sidebar__footer">
        <button type="button" className="sidebar__nav-item" disabled title="Coming soon">
          <Settings size={16} strokeWidth={2} />
          <span>Settings</span>
        </button>
      </div>
    </aside>
  );
}
