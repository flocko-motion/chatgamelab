import type { Plugin, ResolvedConfig } from "vite";

const CLIENT_SCRIPT = `
(function() {
  if (!import.meta.hot) return;

  // Guard against re-entrant calls. When import.meta.hot.send() fails
  // (e.g. Vite dev server is down), the rejection handler internally
  // calls console.error, which would re-enter our override and call
  // send() again — creating an infinite recursion loop that freezes
  // the browser. The flag ensures nested calls go straight to the
  // original console method.
  var sending = false;

  var noop = function() {};
  var methods = ['log', 'info', 'warn', 'error', 'debug'];
  methods.forEach(function(method) {
    var original = console[method].bind(console);
    console[method] = function(...args) {
      original(...args);
      if (sending) return;
      sending = true;
      try {
        var serialized = args.map(function(arg) {
          if (arg instanceof Error) return arg.stack || arg.message;
          if (typeof arg === 'object') {
            try { return JSON.stringify(arg, null, 2); }
            catch { return String(arg); }
          }
          return String(arg);
        }).join(' ');
        // .catch(noop) swallows the rejected promise so the runtime
        // never fires console.error for the rejection, which would
        // re-enter this override and cause infinite recursion.
        Promise.resolve(
          import.meta.hot.send('browser-log', { level: method, message: serialized })
        ).catch(noop).finally(function() { sending = false; });
      } catch { sending = false; }
    };
  });

  window.addEventListener('error', function(e) {
    try {
      Promise.resolve(
        import.meta.hot.send('browser-log', {
          level: 'error',
          message: '[Uncaught] ' + (e.error?.stack || e.message),
        })
      ).catch(noop);
    } catch {}
  });

  window.addEventListener('unhandledrejection', function(e) {
    try {
      var reason = e.reason instanceof Error ? e.reason.stack : String(e.reason);
      Promise.resolve(
        import.meta.hot.send('browser-log', {
          level: 'error',
          message: '[Unhandled Rejection] ' + reason,
        })
      ).catch(noop);
    } catch {}
  });
})();
`;

const LEVEL_COLORS: Record<string, string> = {
  log: "\x1b[36m", // cyan
  info: "\x1b[34m", // blue
  warn: "\x1b[33m", // yellow
  error: "\x1b[31m", // red
  debug: "\x1b[90m", // gray
};
const RESET = "\x1b[0m";
const LABEL = "\x1b[35m[browser]\x1b[0m"; // magenta label

/**
 * Vite plugin that forwards browser console.* calls to the dev server terminal.
 *
 * Enabled by default in dev mode. Set VITE_BROWSER_LOGS=false to disable.
 */
export function browserLogs(): Plugin {
  let disabled = false;

  return {
    name: "browser-logs",
    apply: "serve",

    configResolved(config: ResolvedConfig) {
      const envValue =
        config.env?.VITE_BROWSER_LOGS ?? process.env.VITE_BROWSER_LOGS;
      if (envValue === "false") {
        disabled = true;
      }
    },

    transformIndexHtml: {
      order: "pre",
      handler(html) {
        if (disabled) return html;
        return html.replace(
          "</head>",
          `<script type="module">${CLIENT_SCRIPT}</script>\n</head>`,
        );
      },
    },

    configureServer(server) {
      if (disabled) return;

      server.ws.on(
        "browser-log",
        (data: { level: string; message: string }) => {
          const color = LEVEL_COLORS[data.level] || "";
          const tag = `${color}${data.level.toUpperCase().padEnd(5)}${RESET}`;
          console.log(`${LABEL} ${tag} ${data.message}`);
        },
      );
    },
  };
}
