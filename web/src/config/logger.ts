import { Logger, LogLevel, ConsoleTransport } from '../common/lib/logger';
import env from './env';

// Helper function to parse log level from string
function parseLogLevel(level: string): LogLevel {
  const upperLevel = level.toUpperCase();
  switch (upperLevel) {
    case 'DEBUG':
      return LogLevel.Debug;
    case 'INFO':
      return LogLevel.Info;
    case 'WARNING':
      return LogLevel.Warning;
    case 'ERROR':
      return LogLevel.Error;
    case 'FATAL':
      return LogLevel.Fatal;
    default:
      console.warn(`Invalid log level: ${level}, falling back to INFO`);
      return LogLevel.Info;
  }
}

// Determine log level from environment variables
// LOGGER_LOGLEVEL takes precedence over DEV setting
function getMinLogLevel(): LogLevel {
  // Check for explicit LOGGER_LOGLEVEL environment variable
  const logLevelEnv = import.meta.env.VITE_LOGGER_LOGLEVEL || import.meta.env.LOGGER_LOGLEVEL;
  if (logLevelEnv) {
    return parseLogLevel(logLevelEnv);
  }
  
  // Fall back to DEV setting
  return env.DEV ? LogLevel.Debug : LogLevel.Info;
}

// Global logger instance with environment-aware configuration
export const logger = Logger.create({
  minLevel: getMinLogLevel(),
  transports: [
    new ConsoleTransport(),
    // Future: Add remote transport for production
  ],
});

// Export common scoped loggers for convenience
export const authLogger = logger.withScope('auth');
export const apiLogger = logger.withScope('api');
export const uiLogger = logger.withScope('ui');
export const navigationLogger = logger.withScope('navigation');
