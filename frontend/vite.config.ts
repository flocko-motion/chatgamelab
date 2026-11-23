import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { TanStackRouterVite } from '@tanstack/router-plugin/vite';
import { browserLogs } from './plugins/browser-logs';
import path from 'path';

const envDir = '../'; // Load .env from root directory (single source of truth)

// https://vite.dev/config/
export default defineConfig({
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
    },
  },
  build: {
    // config.gen.ts loads the runtime config with a top-level await
    target: 'es2022',
  },
  optimizeDeps: {
    include: ['react', 'react-dom', '@mantine/core', '@mantine/hooks'],
  },
  // schemaf's server proxies the frontend to port + 2 (schemaf.toml) in development.
  server: {
    port: 7102,
    strictPort: true,
  },
});
