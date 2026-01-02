export const LogLevel = {
  Debug: 0,
  Info: 1,
  Warning: 2,
  Error: 3,
  Fatal: 4,
} as const;

export type LogLevel = typeof LogLevel[keyof typeof LogLevel];

export interface LogEntry {
  level: LogLevel;
  message: string;
  timestamp: Date;
  scope?: string;
  data?: any;
}

export interface LogTransport {
  send(entry: LogEntry): void;
}

export class ConsoleTransport implements LogTransport {
  send(entry: LogEntry): void {
    const timestamp = entry.timestamp.toISOString();
    const scope = entry.scope ? `[${entry.scope}]` : '';
    const prefix = `${timestamp} ${scope}`;
    
    switch (entry.level) {
      case LogLevel.Debug:
        console.debug(prefix, entry.message, entry.data);
        break;
      case LogLevel.Info:
        console.info(prefix, entry.message, entry.data);
        break;
      case LogLevel.Warning:
        console.warn(prefix, entry.message, entry.data);
        break;
      case LogLevel.Error:
        console.error(prefix, entry.message, entry.data);
        break;
      case LogLevel.Fatal:
        console.error(`🔥 FATAL ${prefix}`, entry.message, entry.data);
        break;
    }
  }
}

export class Logger {
  private transports: LogTransport[] = [];
  private minLevel: LogLevel = LogLevel.Info;
  private scope?: string;

  constructor(options?: { scope?: string; minLevel?: LogLevel; transports?: LogTransport[] }) {
    this.scope = options?.scope;
    this.minLevel = options?.minLevel ?? LogLevel.Info;
    this.transports = options?.transports ?? [new ConsoleTransport()];
  }

  private log(level: LogLevel, message: string, data?: any): void {
    if (level < this.minLevel) {
      return;
    }

    const entry: LogEntry = {
      level,
      message,
      timestamp: new Date(),
      scope: this.scope,
      data,
    };

    this.transports.forEach(transport => transport.send(entry));
  }

  debug(message: string, data?: any): void {
    this.log(LogLevel.Debug, message, data);
  }

  info(message: string, data?: any): void {
    this.log(LogLevel.Info, message, data);
  }

  warning(message: string, data?: any): void {
    this.log(LogLevel.Warning, message, data);
  }

  error(message: string, data?: any): void {
    this.log(LogLevel.Error, message, data);
  }

  fatal(message: string, data?: any): void {
    this.log(LogLevel.Fatal, message, data);
  }

  withScope(scope: string): Logger {
    return new Logger({
      scope: this.scope ? `${this.scope}:${scope}` : scope,
      minLevel: this.minLevel,
      transports: this.transports,
    });
  }

  withMinLevel(level: LogLevel): Logger {
    return new Logger({
      scope: this.scope,
      minLevel: level,
      transports: this.transports,
    });
  }

  addTransport(transport: LogTransport): Logger {
    return new Logger({
      scope: this.scope,
      minLevel: this.minLevel,
      transports: [...this.transports, transport],
    });
  }

  static create(options?: { scope?: string; minLevel?: LogLevel; transports?: LogTransport[] }): Logger {
    return new Logger(options);
  }
}

export const logger = Logger.create();
