/**
 * Formats a registered project's absolute path for display: replaces a
 * leading /Users/<name> or /home/<name> with ~. Purely cosmetic — the
 * backend path (the project's actual registered identity) is untouched;
 * callers that need the full path (e.g. a title/tooltip) should keep using
 * the original value alongside this formatted one.
 */
export function formatProjectPath(path: string): string {
  return path.replace(/^\/(Users|home)\/[^/]+/, "~");
}
