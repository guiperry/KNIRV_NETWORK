import * as Sentry from '@sentry/react';

// Environment variable helper that works in both Vite and Jest
const getEnvVar = (key: string): string | undefined => {
  // In Jest environment, disable Sentry completely
  if (typeof process !== 'undefined' && process.env.NODE_ENV === 'test') {
    return undefined;
  }

  // Always try process.env first (works in both Node and Vite)
  if (typeof process !== 'undefined' && process.env && process.env[key]) {
    return process.env[key];
  }

  // In browser environment, we can't access import.meta in Jest
  // So we'll just return undefined for Jest and let Vite handle it at runtime
  return undefined;
};

export const initSentry = () => {
  const dsn = getEnvVar('VITE_SENTRY_DSN');

  if (!dsn) {
    console.warn('Sentry DSN not configured');
    return;
  }

  Sentry.init({
    dsn,
    environment: getEnvVar('VITE_APP_ENV') || 'development',
    integrations: [
      Sentry.browserTracingIntegration(),
    ],

    // Performance monitoring
    tracesSampleRate: getEnvVar('VITE_APP_ENV') === 'production' ? 0.1 : 1.0,

    // Configure error filtering
    beforeSend(event) {
      // Filter out non-critical errors in development
      if (getEnvVar('VITE_APP_ENV') === 'development') {
        // You can add custom filtering logic here
        console.log('Sentry event:', event);
      }
      return event;
    },

    // User feedback
    attachStacktrace: true,

    // Release tracking
    release: getEnvVar('VITE_APP_VERSION') || '1.0.0',
  });
};

export const captureException = (error: Error, context?: Record<string, unknown>) => {
  Sentry.captureException(error, {
    tags: {
      environment: getEnvVar('VITE_APP_ENV'),
      version: getEnvVar('VITE_APP_VERSION'),
    },
    extra: context,
  });
};

export const captureMessage = (message: string, level: Sentry.SeverityLevel = 'info', context?: Record<string, unknown>) => {
  Sentry.captureMessage(message, {
    level,
    tags: {
      environment: getEnvVar('VITE_APP_ENV'),
      version: getEnvVar('VITE_APP_VERSION'),
    },
    extra: context,
  });
};

export const setUser = (user: { id: string; email?: string; username?: string }) => {
  Sentry.setUser(user);
};

export const addBreadcrumb = (message: string, category?: string, level?: Sentry.SeverityLevel) => {
  Sentry.addBreadcrumb({
    message,
    category: category || 'custom',
    level: level || 'info',
  });
};
