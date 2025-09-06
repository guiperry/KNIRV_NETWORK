import * as Sentry from '@sentry/react';
import { BrowserTracing } from '@sentry/tracing';

export const initSentry = () => {
  const dsn = import.meta.env.VITE_SENTRY_DSN;

  if (!dsn) {
    console.warn('Sentry DSN not configured');
    return;
  }

  Sentry.init({
    dsn,
    environment: import.meta.env.VITE_APP_ENV || 'development',
    integrations: [
      new BrowserTracing({
        tracePropagationTargets: [
          'localhost',
          'knirv.network',
          /^https:\/\/.*\.knirv\.network/,
        ],
      }),
    ],

    // Performance monitoring
    tracesSampleRate: import.meta.env.VITE_APP_ENV === 'production' ? 0.1 : 1.0,

    // Release health
    enableTracing: true,

    // Configure error filtering
    beforeSend(event) {
      // Filter out non-critical errors in development
      if (import.meta.env.VITE_APP_ENV === 'development') {
        // You can add custom filtering logic here
        console.log('Sentry event:', event);
      }
      return event;
    },

    // User feedback
    attachStacktrace: true,

    // Release tracking
    release: import.meta.env.VITE_APP_VERSION || '1.0.0',
  });
};

export const captureException = (error: Error, context?: Record<string, unknown>) => {
  Sentry.captureException(error, {
    tags: {
      environment: import.meta.env.VITE_APP_ENV,
      version: import.meta.env.VITE_APP_VERSION,
    },
    extra: context,
  });
};

export const captureMessage = (message: string, level: Sentry.SeverityLevel = 'info', context?: Record<string, unknown>) => {
  Sentry.captureMessage(message, level, {
    tags: {
      environment: import.meta.env.VITE_APP_ENV,
      version: import.meta.env.VITE_APP_VERSION,
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
