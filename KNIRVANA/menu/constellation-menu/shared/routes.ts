import { z } from 'zod';
import { insertGameSettingsSchema, gameSettings } from './schema';

export const errorSchemas = {
  validation: z.object({
    message: z.string(),
    field: z.string().optional(),
  }),
  notFound: z.object({
    message: z.string(),
  }),
  internal: z.object({
    message: z.string(),
  }),
};

export const api = {
  settings: {
    get: {
      method: 'GET' as const,
      path: '/api/settings',
      responses: {
        200: z.custom<typeof gameSettings.$inferSelect>(),
        404: errorSchemas.notFound,
      },
    },
    update: {
      method: 'PATCH' as const,
      path: '/api/settings',
      input: insertGameSettingsSchema.partial(),
      responses: {
        200: z.custom<typeof gameSettings.$inferSelect>(),
        400: errorSchemas.validation,
      },
    },
    reset: {
      method: 'POST' as const,
      path: '/api/settings/reset',
      responses: {
        200: z.custom<typeof gameSettings.$inferSelect>(),
      },
    }
  },
};

export function buildUrl(path: string, params?: Record<string, string | number>): string {
  let url = path;
  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      if (url.includes(`:${key}`)) {
        url = url.replace(`:${key}`, String(value));
      }
    });
  }
  return url;
}
