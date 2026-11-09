import type { ModelsClient, VerifyResponse, ModelStatus, ImageRef } from '@knirv/sdk/models';
import { getKnirvGatewayUrl } from '../config/runtimeConfig';

type ModelsModule = typeof import('@knirv/sdk/models');

/**
 * Text and vision through the KNIRVMODELS daemon, reached over the gateway's
 * `/api/knirvmodels/*` proxy. The SDK's `ModelsClient` owns the wire contract;
 * this class only decides how the three models are combined.
 *
 *   HEART  generates text (decision-aware, may abstain)
 *   HASHER verifies generated text against the evidence index
 *   HEAD   text mirror and vision; its default backend is `none` (503), so it
 *          reports "not ready" until KNIRV_VLM_BACKEND is configured
 *
 * Every method returns `null` (not a guess) when a model is not ready, abstains,
 * or the answer is refuted, so the caller can fall back to the next source.
 */

export interface VerifiedText {
  text: string;
  source: 'heart' | 'head';
  verification: VerifyResponse | null;
}

export class KnirvModelsService {
  // The SDK's models module loads its wasm core on import, so it is loaded on
  // first use rather than when the engine is constructed.
  private clientPromise: Promise<{ client: ModelsClient; sdk: ModelsModule }> | null = null;
  // The SDK module once it has loaded; null while it is unavailable.
  private loadedSdk: ModelsModule | null = null;

  constructor(private readonly injected?: { client: ModelsClient; sdk: ModelsModule }) {
    this.loadedSdk = injected?.sdk ?? null;
  }

  private load(): Promise<{ client: ModelsClient; sdk: ModelsModule }> {
    if (this.injected) return Promise.resolve(this.injected);
    if (!this.clientPromise) {
      this.clientPromise = import('@knirv/sdk/models').then((sdk) => {
        this.loadedSdk = sdk;
        return { sdk, client: new sdk.ModelsClient(`${getKnirvGatewayUrl()}/api/knirvmodels`) };
      });
    }
    return this.clientPromise;
  }

  /**
   * Generates text with HEART and checks it with HASHER. Returns null when HEART
   * is not ready, abstains, or HASHER refutes the text.
   */
  async generateVerifiedText(prompt: string): Promise<VerifiedText | null> {
    try {
      const { client } = await this.load();
      const heart = await client.heartGenerateWithDecision(prompt, true);
      if (!heart.ready || heart.abstained || !heart.text) {
        return this.generateWithHead(prompt);
      }
      const verification = await client.hasherVerify({ text: heart.text, delivery: 'inline', budget_ms: 2000 });
      if (verification.verdict === 'refuted') {
        return null;
      }
      return { text: heart.text, source: 'heart', verification };
    } catch (error) {
      if (this.isNotReady(error)) {
        return this.generateWithHead(prompt);
      }
      console.warn('HEART/HASHER text generation unavailable:', error);
      return null;
    }
  }

  /** HEAD text mirror, used when HEART cannot answer. */
  private async generateWithHead(prompt: string): Promise<VerifiedText | null> {
    try {
      const { client } = await this.load();
      const head = await client.headGenerate({ prompt });
      if (!head.text) return null;
      return { text: head.text, source: 'head', verification: null };
    } catch (error) {
      if (!(this.isNotReady(error))) {
        console.warn('HEAD text mirror unavailable:', error);
      }
      return null;
    }
  }

  /** HEAD vision: describes images. Returns null when HEAD has no vision backend. */
  async describeImages(images: ImageRef[], context?: string): Promise<string | null> {
    try {
      const { client } = await this.load();
      const response = await client.headVision({ operation: 'describe', images, context });
      const described = response.result?.description;
      return typeof described === 'string' ? described : null;
    } catch (error) {
      if (!(this.isNotReady(error))) {
        console.warn('HEAD vision unavailable:', error);
      }
      return null;
    }
  }

  /** Status of every model the daemon serves (HEAD, HEART, HASHER, HANDS). */
  async registry(): Promise<ModelStatus[]> {
    const { client } = await this.load();
    return client.registry();
  }

  /** True when the error is the SDK's NotReadyError. Never triggers a load itself. */
  private isNotReady(error: unknown): boolean {
    return this.loadedSdk !== null && error instanceof this.loadedSdk.NotReadyError;
  }
}
