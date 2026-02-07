import type { Plugin, ResolvedConfig } from 'vite';

const CLIENT_SCRIPT = `
(function() {
  if (!import.meta.hot) return;

  const methods = ['log', 'info', 'warn', 'error', 'debug'];
  methods.forEach(function(method) {
    const original = console[method].bind(console);
    console[method] = function(...args) {
      original(...args);
      try {
        const serialized = args.map(function(arg) {
          if (arg instanceof Error) return arg.stack || arg.message;
          if (typeof arg === 'object') {
            try { return JSON.stringify(arg, null, 2); }
            catch { return String(arg); }
          }
          return String(arg);
        }).join(' ');
        import.meta.hot.send('browser-log', { level: method, message: serialized });
      } catch {}
    };
  });

  window.addEventListener('error', function(e) {
    try {
      import.meta.hot.send('browser-log', {
        level: 'error',
        message: '[Uncaught] ' + (e.error?.stack || e.message),
      });
    } catch {}
  });

  window.addEventListener('unhandledrejection', function(e) {
    try {
      const reason = e.reason instanceof Error ? e.reason.stack : String(e.reason);
      import.meta.hot.send('browser-log', {
        level: 'error',
        message: '[Unhandled Rejection] ' + reason,
      });
    } catch {}
  });
})();
`;

const LEVEL_COLORS: Record<string, string> = {
  log: '\x1b[36m',     // cyan
  info: '\x1b[34m',    // blue
  warn: '\x1b[33m',    // yellow
  error: '\x1b[31m',   // red
  debug: '\x1b[90m',   // gray
};
const RESET = '\x1b[0m';
const LABEL = '\x1b[35m[browser]\x1b[0m'; // magenta label

/**
 * Vite plugin that forwards browser console.* calls to the dev server terminal.
 *
 * Enabled by default in dev mode. Set VITE_BROWSER_LOGS=false to disable.
 */
export function browserLogs(): Plugin {
  let disabled = false;

  return {
    name: 'browser-logs',
    apply: 'serve',

    configResolved(config: ResolvedConfig) {
      const envValue = config.env?.VITE_BROWSER_LOGS ?? process.env.VITE_BROWSER_LOGS;
      if (envValue === 'false') {
        disabled = true;
      }
    },

    transformIndexHtml: {
      order: 'pre',
      handler(html) {
        if (disabled) return html;
        return html.replace(
          '</head>',
          `<script type="module">${CLIENT_SCRIPT}</script>\n</head>`,
        );
      },
    },

    configureServer(server) {
      if (disabled) return;

      server.ws.on('browser-log', (data: { level: string; message: string }) => {
        const color = LEVEL_COLORS[data.level] || '';
        const tag = `${color}${data.level.toUpperCase().padEnd(5)}${RESET}`;
        console.log(`${LABEL} ${tag} ${data.message}`);
      });
    },
  };
}
