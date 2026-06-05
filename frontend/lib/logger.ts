import pino from 'pino';

const isDev = process.env.NODE_ENV === 'development';
const isServer = typeof window === 'undefined';

const serverLogger = isServer
  ? pino({
      level: isDev ? 'debug' : 'info',
      transport: isDev
        ? {
            target: 'pino-pretty',
            options: {
              colorize: true,
              messageFormat: '[{service}] {msg}',
              ignore: 'pid,hostname',
            },
          }
        : undefined,
      base: { service: 'nextjs-frontend' },
    })
  : null;

const clientLogger = {
  info: (msg: string, ...args: any[]) =>
    isDev ? console.log(`%c[Frontend-Client] ${msg}`, 'color: #4CAF50', ...args) : null,
  warn: (msg: string, ...args: any[]) =>
    isDev ? console.warn(`%c[Frontend-Client] ${msg}`, 'color: #FF9800', ...args) : null,
  error: (msg: string, ...args: any[]) =>
    console.error(`%c[Frontend-Client] ${msg}`, 'color: #F44336', ...args),
};

export const logger = {
  info: (msg: string, ...args: any[]) => {
    if (isServer && serverLogger) serverLogger.info(msg, ...args);
    else clientLogger.info(msg, ...args);
  },
  warn: (msg: string, ...args: any[]) => {
    if (isServer && serverLogger) serverLogger.warn(msg, ...args);
    else clientLogger.warn(msg, ...args);
  },
  error: (msg: string, ...args: any[]) => {
    if (isServer && serverLogger) serverLogger.error(msg, ...args);
    else clientLogger.error(msg, ...args);
  },
};

export { serverLogger };
