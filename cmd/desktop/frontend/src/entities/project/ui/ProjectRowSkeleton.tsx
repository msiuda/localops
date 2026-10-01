const pulse = "animate-pulse rounded-sm bg-border-strong";

/** A loading placeholder shaped like a ProjectRow, shown while Overview loads. */
export function ProjectRowSkeleton() {
  return (
    <div
      className="border-border bg-surface flex items-center justify-between gap-4 rounded-md border px-4 py-3"
      aria-hidden="true"
    >
      <div className="flex flex-1 flex-col gap-2">
        <div className={`h-3 w-35 ${pulse}`} />
        <div className={`h-2.5 w-55 ${pulse}`} />
      </div>
      <div className={`h-5 w-22 rounded-full ${pulse}`} />
    </div>
  );
}
