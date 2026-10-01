import type { ReactNode } from "react";
import { Sidebar } from "./Sidebar";

export function AppShell({ children }: { children: ReactNode }) {
  return (
    <div className="bg-bg flex h-screen w-screen">
      <Sidebar />
      <main className="h-full min-w-0 flex-1">{children}</main>
    </div>
  );
}
