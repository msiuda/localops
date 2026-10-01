import { useMutation } from "@tanstack/react-query";
import { projectApi, useInvalidateProjectsOverview } from "@/entities/project";

/**
 * Drives the Add Project flow: open the native directory picker, and if the
 * user picked something (an empty path means they cancelled — do nothing),
 * register it through the same shared registration behavior the CLI uses.
 * On success, invalidates the Projects overview query instead of manually
 * duplicating server-state sync with useState.
 */
export function useAddProjectMutation() {
  const invalidateOverview = useInvalidateProjectsOverview();

  return useMutation({
    mutationFn: async () => {
      const path = await projectApi.pickProjectDirectory();
      if (!path) {
        return null;
      }
      await projectApi.addProject(path);
      return path;
    },
    onSuccess: (path) => {
      if (path) {
        invalidateOverview();
      }
    },
  });
}
