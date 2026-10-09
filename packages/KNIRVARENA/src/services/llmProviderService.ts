// LLM Provider Service — chat through KNIRVSERVER (arena_fixes.md §3.5).
//
// The browser holds no provider keys. Each call goes to `POST /api/llm/complete`,
// which uses the signed-in user's own key from KNIRVSERVER secret storage, or
// the server's local llama.cpp model (no key) for the "adaline" option, which
// is now KNIRV's local model rather than an external gateway.
import type { LLMProvider, ChatMessage, ChatResponse } from '../types/chatBrain';
import { authToken, completeOnServer, DEFAULT_MODELS, type ServerLLMProvider } from './serverLLM';

export interface ProviderConfig {
  model?: string;
  temperature?: number;
  maxTokens?: number;
}

export interface LLMProviderOptions {
  gemini?: ProviderConfig;
  openai?: ProviderConfig;
  deepseek?: ProviderConfig;
}

const SERVER_PROVIDER: Record<LLMProvider, ServerLLMProvider> = {
  gemini: 'gemini',
  openai: 'openai',
  deepseek: 'deepseek',
  adaline: 'llama',
};

const ALL_PROVIDERS: LLMProvider[] = ['adaline', 'gemini', 'openai', 'deepseek'];

export class LLMProviderService {
  constructor(private readonly options: LLMProviderOptions = {}) {}

  async chat(message: string, provider: LLMProvider, history?: ChatMessage[]): Promise<ChatResponse> {
    return this.chatWithOptions(message, provider, {}, history);
  }

  async chatWithOptions(
    message: string,
    provider: LLMProvider,
    options: ProviderConfig,
    history?: ChatMessage[],
  ): Promise<ChatResponse> {
    const serverProvider = SERVER_PROVIDER[provider];
    if (!serverProvider) throw new Error(`Unsupported provider: ${provider}`);
    const configured = provider === 'adaline' ? undefined : this.options[provider];
    const model = serverProvider === 'llama' ? undefined : options.model ?? configured?.model ?? DEFAULT_MODELS[serverProvider];
    const text = await completeOnServer({
      provider: serverProvider,
      model,
      messages: this.buildChatMessages(message, history),
      maxTokens: options.maxTokens ?? configured?.maxTokens,
    });
    return { text, provider, metadata: { model: model ?? 'knirv-local-llama', via: 'knirvserver' } };
  }

  private buildChatMessages(message: string, history?: ChatMessage[]): Array<{ role: 'user' | 'assistant'; content: string }> {
    const messages = (history ?? []).slice(-20).map((msg) => ({
      role: (msg.type === 'user' ? 'user' : 'assistant') as 'user' | 'assistant',
      content: msg.text,
    }));
    messages.push({ role: 'user', content: message });
    return messages;
  }

  /**
   * Whether a provider can be tried. Keys live on the server, so availability
   * here only means "signed in"; a missing key is reported by the server with
   * a prompt to add it in menu → Provider Keys.
   */
  isProviderAvailable(provider: LLMProvider): boolean {
    return Boolean(authToken()) && provider in SERVER_PROVIDER;
  }

  getAvailableProviders(): LLMProvider[] {
    return ALL_PROVIDERS.filter((p) => this.isProviderAvailable(p));
  }

  getProviderName(provider: LLMProvider): string {
    const names: Record<LLMProvider, string> = {
      gemini: 'Google Gemini',
      openai: 'OpenAI GPT',
      deepseek: 'DeepSeek',
      adaline: 'KNIRV local model',
    };
    return names[provider];
  }

  getProviderIcon(provider: LLMProvider): string {
    const icons: Record<LLMProvider, string> = { gemini: '🤖', openai: '🧠', deepseek: '🔍', adaline: '⚡' };
    return icons[provider];
  }

  /** The local model is always offered when signed in. */
  isGatewayReady(): boolean {
    return Boolean(authToken());
  }

  getGatewayStatus(): { adalineReady: boolean; providers: { gemini: boolean; openai: boolean; deepseek: boolean } } {
    const signedIn = Boolean(authToken());
    return { adalineReady: signedIn, providers: { gemini: signedIn, openai: signedIn, deepseek: signedIn } };
  }
}

let llmProviderServiceInstance: LLMProviderService | null = null;

export const getLLMProviderService = (): LLMProviderService => {
  llmProviderServiceInstance ??= new LLMProviderService();
  return llmProviderServiceInstance;
};

export const createLLMProviderService = (options?: LLMProviderOptions): LLMProviderService => new LLMProviderService(options);

export default getLLMProviderService;
