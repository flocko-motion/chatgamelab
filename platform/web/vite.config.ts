import { defineConfig, loadEnv, type ProxyOptions } from 'vite';
import react from '@vitejs/plugin-react';
import { TanStackRouterVite } from '@tanstack/router-plugin/vite';
import { browserLogs } from './plugins/browser-logs';
import path from 'path';

const envDir = '../../'; // Load .env from root directory (single source of truth)

/**
 * The playback-engine tells its parent page about a new session only when both
 * share an origin. The API runs on its own port in development, so the dev
 * server serves the engine's routes on the page's origin
 * (src/features/play/playback.ts). The Host header is kept, so the engine's
 * websocket accepts the page's Origin.
 */
function engineProxy(mode: string): Record<string, ProxyOptions> {
  const apiBase = loadEnv(mode, envDir, 'VITE_').VITE_API_BASE_URL;
  // A relative API base already shares the page's origin.
  if (!apiBase || !URL.canParse(apiBase)) return {};
  const api = new URL(apiBase);
  const path = `${api.pathname.replace(/\/$/, '')}/engine`;
  return { [path]: { target: api.origin, changeOrigin: false, ws: true } };
}

// https://vite.dev/config/
export default defineConfig(({ mode }) => ({
  envDir,
  // Use relative paths so assets resolve correctly with a runtime-patched <base> tag
  base: './',
  plugins: [
    TanStackRouterVite({
      routesDirectory: './src/routes',
      generatedRouteTree: './src/routeTree.gen.ts',
    }),
    react(),
    browserLogs(),
  ],
  resolve: {
    dedupe: ['react', 'react-dom'],
    alias: {
      '@': path.resolve(__dirname, './src'),
      '@api': path.resolve(__dirname, './src/api'),
      '@common': path.resolve(__dirname, './src/common'),
      '@config': path.resolve(__dirname, './src/config'),
      '@features': path.resolve(__dirname, './src/features'),
      '@i18n': path.resolve(__dirname, './src/i18n'),
      '@components': path.resolve(__dirname, './src/common/components'),
      '@hooks': path.resolve(__dirname, './src/common/hooks'),
      '@lib': path.resolve(__dirname, './src/common/lib'),
      '@types': path.resolve(__dirname, './src/common/types'),
      '@version': path.resolve(__dirname, './src/version.js'),
    },
  },
  optimizeDeps: {
    include: ['react', 'react-dom', '@mantine/core', '@mantine/hooks'],
  },
  server: {
    proxy: engineProxy(mode),
  },
}));
