/// <reference types="@cloudflare/workers-types" />

/**
 * Welcome to Cloudflare Workers! This is your first worker.
 *
 * - Run `npm install -g wrangler` in your terminal to install wrangler
 * - Run `wrangler dev` in this directory to start a development server
 * - Run `wrangler deploy` to publish your worker
 *
 * Learn more at https://developers.cloudflare.com/workers/
 */

import { getAssetFromKV, NotFoundError } from '@cloudflare/kv-asset-handler';

export default {
  async fetch(request: Request, env: any, ctx: ExecutionContext): Promise<Response> {
    try {
      // Use the asset handler to serve static files
      return await getAssetFromKV({
        request,
        waitUntil: (promise: Promise<any>) => ctx.waitUntil(promise),
      }, { ASSET_NAMESPACE: env.__STATIC_CONTENT, ASSET_MANIFEST: env.__STATIC_CONTENT_MANIFEST });
    } catch (e) {
      if (e instanceof NotFoundError) {
        return new Response('Not Found', { status: 404 });
      }
      return new Response('Internal Server Error', { status: 500 });
    }
  },
};