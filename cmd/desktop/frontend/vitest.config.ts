import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

// A dedicated Vitest config (rather than reusing vite.config.ts) so tests
// never need the TanStack Router codegen plugin or the Wails dev-server
// plugin — neither makes sense outside a real Vite dev/build run.
export default defineConfig({
  resolve: {
    tsconfigPaths: true,
  },
  plugins: [react()],
  test: {
    // Enables RTL's automatic post-test DOM cleanup (it registers via a
    // global `afterEach`, which only exists when this is on).
    globals: true,
    environment: "jsdom",
    setupFiles: ["./vitest.setup.ts"],
    css: false,
    coverage: {
      provider: "v8",
      reporter: ["text", "html"],
    },
  },
});
