import { FolderPlus } from "lucide-react";
import { Button } from "@/shared/ui/atoms/Button";
import { useAddProjectMutation } from "../model/useAddProjectMutation";

interface AddProjectButtonProps {
  className?: string;
}

/** The Add Project trigger, reused in the Projects header and empty state. */
export function AddProjectButton({ className }: AddProjectButtonProps) {
  const mutation = useAddProjectMutation();

  return (
    <div className="flex flex-col items-end gap-2">
      <Button className={className} onClick={() => mutation.mutate()} disabled={mutation.isPending}>
        <FolderPlus size={14} strokeWidth={2.2} />
        {mutation.isPending ? "Adding…" : "Add Project"}
      </Button>

      {mutation.isError && (
        <div className="border-danger-bg bg-danger-bg text-danger text-body flex items-center gap-3 rounded-sm border px-3.5 py-2">
          <span>
            Couldn't add project:{" "}
            {mutation.error instanceof Error ? mutation.error.message : String(mutation.error)}
          </span>
          <button
            type="button"
            className="text-danger text-micro flex-shrink-0 font-semibold underline"
            onClick={() => mutation.reset()}
          >
            Dismiss
          </button>
        </div>
      )}
    </div>
  );
}
