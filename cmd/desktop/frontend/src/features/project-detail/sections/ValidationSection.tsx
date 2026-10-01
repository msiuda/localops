import { ShieldCheck } from "lucide-react";

/**
 * Validation is not runnable from the desktop app yet. A compact
 * informational panel — not a full-width empty state — since this is a
 * real, expected state, not an absence of content. No command list, no
 * speculative preview of what would run: LocalOps Validation can execute
 * real project commands, and desktop execution will be its own deliberate
 * feature built on the real internal/validation package, not recreated here.
 */
export function ValidationSection() {
  return (
    <div className="border-border bg-surface flex max-w-125 gap-3 rounded-md border p-3.5">
      <ShieldCheck
        className="text-text-muted mt-0.5 flex-shrink-0"
        size={18}
        strokeWidth={1.75}
        aria-hidden="true"
      />
      <div className="flex flex-col gap-1.5">
        <p className="text-text-primary m-0 text-[12.5px] font-semibold">
          Validation isn't runnable from the desktop yet
        </p>
        <p className="text-text-muted m-0 text-xs leading-relaxed">
          LocalOps can run project tests, builds, linting, and related checks from the CLI. Desktop execution
          will be added as its own dedicated, deliberate workflow — nothing runs merely by opening this
          project.
        </p>
        <code className="border-border bg-bg text-text-secondary mt-0.5 w-fit rounded-sm border px-2 py-0.5 font-mono text-[11px]">
          localops validate &lt;path&gt;
        </code>
      </div>
    </div>
  );
}
