import { Boxes, Stethoscope, ShieldCheck, KeyRound, Settings, type LucideIcon } from "lucide-react";
import { Link, useRouterState } from "@tanstack/react-router";
import { cn } from "@/shared/lib/cn";
import { LocalOpsMark } from "./LocalOpsMark";

interface NavItemConfig {
  key: string;
  label: string;
  icon: LucideIcon;
}

const DISABLED_NAV_ITEMS: NavItemConfig[] = [
  { key: "doctor", label: "Doctor", icon: Stethoscope },
  { key: "validation", label: "Validation", icon: ShieldCheck },
  { key: "environment", label: "Environment", icon: KeyRound },
];

const navItemClasses =
  "flex w-full items-center gap-2.5 rounded-sm px-2 py-1.5 text-[12.5px] font-medium text-text-secondary";

/**
 * The application sidebar: window controls, wordmark, primary navigation,
 * and a pinned Settings entry. Doctor / Validation / Environment are
 * intentionally non-functional global-navigation placeholders in this
 * milestone — they exist to establish LocalOps's information architecture,
 * not to be usable yet. Projects is the one enabled top-level area, and
 * stays active/selected whether the app is showing the Projects list or a
 * project's Detail view.
 */
export function Sidebar() {
  const pathname = useRouterState({ select: (state) => state.location.pathname });
  const isProjectsActive = pathname === "/" || pathname.startsWith("/project");

  return (
    <aside className="border-border bg-sidebar flex h-full w-55 min-w-55 flex-col border-r">
      {/* pt-11 reserves the native traffic-light safe area (MacTitleBarHidden's
          AppKit buttons occupy roughly the first 28-30px) plus a deliberate
          breathing-room gap, so the brand row starts clear of the native
          buttons instead of crowding them. */}
      <div className="drag-region flex flex-col px-3.5 pt-11 pb-4">
        <div className="flex items-center gap-2 pl-0.5">
          <span className="bg-accent flex h-5.5 w-5.5 items-center justify-center rounded-md text-white">
            <LocalOpsMark size={13} />
          </span>
          <span className="text-text-primary text-[13px] font-semibold tracking-[0.01em]">LocalOps</span>
        </div>
      </div>

      <nav className="flex flex-col gap-0.5 px-2.5 py-1">
        <Link
          to="/"
          className={cn(
            navItemClasses,
            "hover:bg-surface-hover hover:text-text-primary focus-visible:outline-accent cursor-pointer focus-visible:outline-2 focus-visible:-outline-offset-2",
            isProjectsActive && "bg-accent-bg text-accent hover:bg-accent-bg hover:text-accent",
          )}
        >
          <Boxes size={16} strokeWidth={2} />
          <span>Projects</span>
        </Link>

        {DISABLED_NAV_ITEMS.map((item) => (
          <button
            key={item.key}
            type="button"
            className={cn(navItemClasses, "opacity-45")}
            disabled
            title="Coming soon"
          >
            <item.icon size={16} strokeWidth={2} />
            <span>{item.label}</span>
          </button>
        ))}
      </nav>

      <div className="border-border mt-auto border-t p-2.5">
        <button type="button" className={cn(navItemClasses, "opacity-45")} disabled title="Coming soon">
          <Settings size={16} strokeWidth={2} />
          <span>Settings</span>
        </button>
      </div>
    </aside>
  );
}
