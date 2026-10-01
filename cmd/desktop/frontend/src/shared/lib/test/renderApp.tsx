import { render } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RouterProvider, createRouter, createMemoryHistory } from "@tanstack/react-router";
import { routeTree } from "../../../routeTree.gen";

/**
 * Renders the real app router (real routes, real navigation) against an
 * in-memory history, so tests exercise actual routing behavior instead of
 * a component in isolation. Only the Wails API boundary
 * (entities/project/api/projectApi) should be mocked by the test itself —
 * this helper never touches Wails.
 */
export function renderApp(initialPath: string) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const router = createRouter({
    routeTree,
    history: createMemoryHistory({ initialEntries: [initialPath] }),
  });

  const utils = render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  );

  return { ...utils, router, queryClient };
}
