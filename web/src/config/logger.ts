import { Logger, LogLevel, ConsoleTransport } from '../common/lib/logger';
import env from './env';

// Global logger instance with environment-aware configuration
export const logger = Logger.create({
  minLevel: env.DEV ? LogLevel.Debug : LogLevel.Info,
  transports: [
    new ConsoleTransport(),
    // Future: Add remote transport for production
  ],
});

// Export common scoped loggers for convenience
export const authLogger = logger.withScope('auth');
export const apiLogger = logger.withScope('api');
export const uiLogger = logger.withScope('ui');
