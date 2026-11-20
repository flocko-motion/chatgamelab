import { createRouter } from '@tanstack/react-router';
import { routeTree } from '../routeTree.gen';
import { config } from './env';

// Normalize basepath: ensure it has leading slash, no trailing slash (unless root)
function normalizeBasepath(url: string | undefined): string {
  if (!url || url === '/') return '/';
  // Extract path from full URL if needed
  let path = url;
  try {
    const parsed = new URL(url);
    path = parsed.pathname;
  } catch {
    // Not a full URL, use as-is
  }
  // Ensure leading slash, remove trailing slash
  if (!path.startsWith('/')) path = '/' + path;
  if (path.length > 1 && path.endsWith('/')) path = path.slice(0, -1);
  return path;
}

export const router = createRouter({
  routeTree,
  basepath: normalizeBasepath(config.PUBLIC_URL),
});

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
}
