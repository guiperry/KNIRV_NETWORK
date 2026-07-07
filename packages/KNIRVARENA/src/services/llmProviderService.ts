// LLM Provider Service - Multi-LLM Support with Adaline Gateway
import { GoogleGenerativeAI, GenerativeModel, type Content } from '@google/generative-ai';
import type { LLMProvider, ChatMessage, ChatResponse } from '../types/chatBrain';

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

export class LLMProviderService {
  private gemini: GoogleGenerativeAI | null = null;
  private geminiModel: GenerativeModel | null = null;
  private options: LLMProviderOptions;
  private adalineInitialized: boolean = false;

  constructor(options: LLMProviderOptions = {}) {
    this.options = options;
    this.initializeProviders();
  }

  private initializeProviders(): void {
    this.initializeGemini();
    this.initializeAdaline();
  }

  private initializeGemini(): void {
    const geminiKey = import.meta.env.VITE_GOOGLE_API_KEY;
    if (geminiKey) {
      try {
        this.gemini = new GoogleGenerativeAI(geminiKey);
        const modelName = this.options.gemini?.model || 'gemini-1.5-pro';
        this.geminiModel = this.gemini.getGenerativeModel({ model: modelName });
      } catch (error) {
        console.error('Failed to initialize Gemini:', error);
      }
    }
  }

  private initializeAdaline(): void {
    const adalineKey = import.meta.env.VITE_ADALINE_API_KEY;
    if (adalineKey) {
      this.adalineInitialized = true;
      console.log('Adaline Gateway: Using multi-model routing via ERGO');
    }
  }

  async chat(
    message: string,
    provider: LLMProvider,
    history?: ChatMessage[]
  ): Promise<ChatResponse> {
    switch (provider) {
      case 'gemini':
        return this.geminiChat(message, history);
      case 'openai':
        return this.openAIChat(message, history);
      case 'deepseek':
        return this.deepseekChat(message, history);
      case 'adaline':
        return this.adalineChat(message, history);
      default:
        throw new Error(`Unsupported provider: ${provider}`);
    }
  }

  async chatWithOptions(
    message: string,
    provider: LLMProvider,
    options: ProviderConfig,
    history?: ChatMessage[]
  ): Promise<ChatResponse> {
    switch (provider) {
      case 'gemini':
        return this.geminiChat(message, history);
      case 'openai':
        return this.openAIChatWithOptions(message, options, history);
      case 'deepseek':
        return this.deepseekChat(message, history);
      case 'adaline':
        return this.adalineChatWithOptions(message, options, history);
      default:
        throw new Error(`Unsupported provider: ${provider}`);
    }
  }

  private async geminiChat(message: string, history?: ChatMessage[]): Promise<ChatResponse> {
    if (!this.geminiModel) {
      throw new Error('Gemini not initialized. Please check your API key.');
    }

    try {
      const chatHistory = this.buildGeminiHistory(history);

      if (chatHistory.length > 0) {
        const chat = this.geminiModel.startChat({
          history: chatHistory,
        });

        const result = await chat.sendMessage(message);
        const response = await result.response;

        return {
          text: response.text(),
          provider: 'gemini',
          metadata: {
            model: this.options.gemini?.model || 'gemini-1.5-pro',
          },
        };
      }

      const result = await this.geminiModel.generateContent(message);
      const response = await result.response;

      return {
        text: response.text(),
        provider: 'gemini',
        metadata: {
          model: this.options.gemini?.model || 'gemini-1.5-pro',
        },
      };
    } catch (error) {
      console.error('Gemini chat error:', error);
      throw new Error(`Gemini error: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  private buildGeminiHistory(history?: ChatMessage[]): Content[] {
    if (!history || history.length === 0) return [];

    return history.map((msg) => ({
      role: msg.type === 'user' ? 'user' : 'model',
      parts: [{ text: msg.text }],
    }));
  }

  private async openAIChat(message: string, history?: ChatMessage[]): Promise<ChatResponse> {
    return this.openAIChatWithOptions(message, this.options.openai || {}, history);
  }

  private async openAIChatWithOptions(
    message: string,
    options: ProviderConfig,
    history?: ChatMessage[]
  ): Promise<ChatResponse> {
    const openAIKey = import.meta.env.VITE_OPENAI_API_KEY;

    if (!openAIKey) {
      throw new Error('OpenAI API key not configured. Set VITE_OPENAI_API_KEY.');
    }

    try {
      const messages = this.buildChatMessages(message, history);
      const model = options.model || 'gpt-4-turbo';
      const temperature = options.temperature ?? 0.7;
      const maxTokens = options.maxTokens ?? 4096;

      const response = await fetch('https://api.openai.com/v1/chat/completions', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${openAIKey}`,
        },
        body: JSON.stringify({
          model,
          messages,
          temperature,
          max_tokens: maxTokens,
        }),
      });

      if (!response.ok) {
        throw new Error(`OpenAI API error: ${response.statusText}`);
      }

      const data = await response.json();

      return {
        text: data.choices[0]?.message?.content || '',
        provider: 'openai',
        metadata: {
          model,
          usage: data.usage,
        },
      };
    } catch (error) {
      console.error('OpenAI chat error:', error);
      throw new Error(`OpenAI error: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  private async deepseekChat(message: string, history?: ChatMessage[]): Promise<ChatResponse> {
    const deepseekKey = import.meta.env.VITE_DEEPSEEK_API_KEY;

    if (!deepseekKey) {
      throw new Error('DeepSeek API key not configured. Set VITE_DEEPSEEK_API_KEY.');
    }

    try {
      const messages = this.buildChatMessages(message, history);

      const response = await fetch('https://api.deepseek.com/v1/chat/completions', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${deepseekKey}`,
        },
        body: JSON.stringify({
          model: 'deepseek-chat',
          messages,
          temperature: 0.7,
        }),
      });

      if (!response.ok) {
        throw new Error(`DeepSeek API error: ${response.statusText}`);
      }

      const data = await response.json();

      return {
        text: data.choices[0]?.message?.content || '',
        provider: 'deepseek',
        metadata: {
          model: 'deepseek-chat',
          usage: data.usage,
        },
      };
    } catch (error) {
      console.error('DeepSeek chat error:', error);
      throw new Error(`DeepSeek error: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  private async adalineChat(message: string, history?: ChatMessage[]): Promise<ChatResponse> {
    return this.adalineChatWithOptions(message, this.options.openai || {}, history);
  }

  private async adalineChatWithOptions(
    message: string,
    options: ProviderConfig,
    history?: ChatMessage[]
  ): Promise<ChatResponse> {
    const adalineKey = import.meta.env.VITE_ADALINE_API_KEY;

    if (!adalineKey) {
      return this.fallbackToLocalModel(message, history);
    }

    try {
      const messages = this.buildChatMessages(message, history);
      const model = options.model || 'gpt-4-turbo';
      const temperature = options.temperature ?? 0.7;
      const maxTokens = options.maxTokens ?? 4096;

      const response = await fetch('https://api.adaline.ai/v1/chat/completions', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${adalineKey}`,
        },
        body: JSON.stringify({
          model,
          messages,
          temperature,
          max_tokens: maxTokens,
          routing: {
            enabled: true,
            strategy: 'latency-aware',
          },
        }),
      });

      if (!response.ok) {
        console.warn('Adaline API error, falling back to local processing');
        return this.fallbackToLocalModel(message, history);
      }

      const data = await response.json();

      return {
        text: data.choices[0]?.message?.content || '',
        provider: 'adaline',
        metadata: {
          model,
          routing: data.routing || { provider: 'unknown' },
          gateway: 'adaline',
        },
      };
    } catch (error) {
      console.warn('Adaline Gateway error, using local fallback:', error);
      return this.fallbackToLocalModel(message, history);
    }
  }

  private async fallbackToLocalModel(
    message: string,
    history?: ChatMessage[]
  ): Promise<ChatResponse> {
    if (this.geminiModel) {
      try {
        const fullMessage = this.buildContextMessage(message, history);
        const result = await this.geminiModel.generateContent(fullMessage);
        const response = await result.response;

        return {
          text: response.text(),
          provider: 'adaline',
          metadata: {
            model: 'gemini-1.5-pro (fallback)',
            fallback: true,
          },
        };
      } catch (geminiError) {
        console.error('Fallback to Gemini failed:', geminiError);
      }
    }

    return {
      text: `[Local processing] Processed: ${message.substring(0, 100)}...`,
      provider: 'adaline',
      metadata: {
        model: 'local-fallback',
        processed: true,
      },
    };
  }

  private buildContextMessage(message: string, history?: ChatMessage[]): string {
    let context = '';

    if (history && history.length > 0) {
      context += 'Previous conversation:\n';
      for (const msg of history.slice(-5)) {
        context += `${msg.type === 'user' ? 'User' : 'Assistant'}: ${msg.text}\n`;
      }
      context += '\n';
    }

    context += `Current request: ${message}`;
    return context;
  }

  private buildChatMessages(
    message: string,
    history?: ChatMessage[]
  ): Array<{ role: string; content: string }> {
    const messages: Array<{ role: string; content: string }> = [];

    if (history && history.length > 0) {
      history.forEach((msg) => {
        messages.push({
          role: msg.type === 'user' ? 'user' : 'assistant',
          content: msg.text,
        });
      });
    }

    messages.push({
      role: 'user',
      content: message,
    });

    return messages;
  }

  isProviderAvailable(provider: LLMProvider): boolean {
    switch (provider) {
      case 'gemini':
        return this.geminiModel !== null;
      case 'openai':
        return !!import.meta.env.VITE_OPENAI_API_KEY;
      case 'adaline':
        return this.adalineInitialized || !!import.meta.env.VITE_OPENAI_API_KEY || !!import.meta.env.VITE_GOOGLE_API_KEY;
      case 'deepseek':
        return !!import.meta.env.VITE_DEEPSEEK_API_KEY;
      default:
        return false;
    }
  }

  getAvailableProviders(): LLMProvider[] {
    const providers: LLMProvider[] = [];

    if (this.isProviderAvailable('gemini')) providers.push('gemini');
    if (this.isProviderAvailable('openai')) providers.push('openai');
    if (this.isProviderAvailable('deepseek')) providers.push('deepseek');
    if (this.isProviderAvailable('adaline')) providers.push('adaline');

    return providers;
  }

  getProviderName(provider: LLMProvider): string {
    const names: Record<LLMProvider, string> = {
      gemini: 'Google Gemini',
      openai: 'OpenAI GPT',
      deepseek: 'DeepSeek',
      adaline: 'Adaline Multi-Model',
    };

    return names[provider];
  }

  getProviderIcon(provider: LLMProvider): string {
    const icons: Record<LLMProvider, string> = {
      gemini: '🤖',
      openai: '🧠',
      deepseek: '🔍',
      adaline: '⚡',
    };

    return icons[provider];
  }

  isGatewayReady(): boolean {
    return this.adalineInitialized;
  }

  getGatewayStatus(): {
    adalineReady: boolean;
    providers: {
      gemini: boolean;
      openai: boolean;
      deepseek: boolean;
    };
  } {
    return {
      adalineReady: this.adalineInitialized,
      providers: {
        gemini: this.geminiModel !== null,
        openai: !!import.meta.env.VITE_OPENAI_API_KEY,
        deepseek: !!import.meta.env.VITE_DEEPSEEK_API_KEY,
      },
    };
  }
}

let llmProviderServiceInstance: LLMProviderService | null = null;

export const getLLMProviderService = (): LLMProviderService => {
  if (!llmProviderServiceInstance) {
    llmProviderServiceInstance = new LLMProviderService();
  }
  return llmProviderServiceInstance;
};

export const createLLMProviderService = (options?: LLMProviderOptions): LLMProviderService => {
  llmProviderServiceInstance = new LLMProviderService(options);
  return llmProviderServiceInstance;
};

export default getLLMProviderService;
