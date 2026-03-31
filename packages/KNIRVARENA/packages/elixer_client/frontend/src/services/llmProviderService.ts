// LLM Provider Service - Multi-LLM Support with Adaline Gateway
import { GoogleGenerativeAI, GenerativeModel, type Content } from '@google/generative-ai';
import { Gateway } from '@adaline/gateway';
import type { LLMProvider, ChatMessage, ChatResponse } from '../types/chatBrain';

export class LLMProviderService {
  private gemini: GoogleGenerativeAI | null = null;
  private geminiModel: GenerativeModel | null = null;
  private adalineGateway: Gateway | null = null;

  constructor() {
    this.initializeProviders();
  }

  private initializeProviders(): void {
    // Initialize Google Gemini
    const geminiKey = import.meta.env.VITE_GOOGLE_API_KEY;
    if (geminiKey) {
      try {
        this.gemini = new GoogleGenerativeAI(geminiKey);
        this.geminiModel = this.gemini.getGenerativeModel({ model: 'gemini-1.5-pro' });
      } catch (error) {
        console.error('Failed to initialize Gemini:', error);
      }
    }

    // Initialize Adaline Gateway
    this.initializeAdalineGateway();
  }

  private async initializeAdalineGateway(): Promise<void> {
    const adalineKey = import.meta.env.VITE_ADALINE_API_KEY;
    
    if (!adalineKey) {
      console.warn('Adaline API key not found in environment variables');
      return;
    }

    try {
      // Initialize Adaline Gateway with correct constructor
      this.adalineGateway = new Gateway({
        enableProxyAgent: true,
        queueOptions: {
          timeout: 30000,
          retry: {
            initialDelay: 1000,
            exponentialFactor: 2
          },
          maxConcurrentTasks: 5,
          retryCount: 3
        }
      });

      console.log('Adaline Gateway initialized successfully');
    } catch (error) {
      console.error('Failed to initialize Adaline Gateway:', error);
      this.adalineGateway = null;
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

  private async geminiChat(message: string, history?: ChatMessage[]): Promise<ChatResponse> {
    if (!this.geminiModel) {
      throw new Error('Gemini not initialized. Please check your API key.');
    }

    try {
      // Build chat history for Gemini
      const chatHistory = this.buildGeminiHistory(history);

      // Start a chat session if we have history
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
            model: 'gemini-1.5-pro',
          },
        };
      }

      // Single message without history
      const result = await this.geminiModel.generateContent(message);
      const response = await result.response;

      return {
        text: response.text(),
        provider: 'gemini',
        metadata: {
          model: 'gemini-1.5-pro',
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

  private async openAIChat(_message: string, _history?: ChatMessage[]): Promise<ChatResponse> {
    // Use Adaline Gateway for OpenAI instead of direct API
    if (!this.adalineGateway) {
      throw new Error('Adaline Gateway not initialized for OpenAI provider.');
    }

    try {
      const messages = this.buildChatMessages(_message, _history);
      
      // TODO: Fix Gateway interface - temporarily disabled
      // Use Adaline to call OpenAI
      // const response = await this.adalineGateway.openai.chat.completions.create({
      //   model: 'gpt-4-turbo',
      //   messages,
      //   temperature: 0.7,
      //   maxTokens: 4096
      // });
      const response = { choices: [{ message: { content: 'OpenAI response via Adaline (placeholder)' } }] };

      return {
        text: response.choices[0]?.message?.content || '',
        provider: 'openai',
        metadata: {
          model: 'gpt-4-turbo',
          usage: { prompt_tokens: 0, completion_tokens: 0, total_tokens: 0 },
          gateway: 'adaline'
        },
      };
    } catch (error) {
      console.error('OpenAI via Adaline error:', error);
      throw new Error(`OpenAI error: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  private async deepseekChat(message: string, history?: ChatMessage[]): Promise<ChatResponse> {
    const deepseekKey = import.meta.env.VITE_DEEPSEEK_API_KEY;

    if (!deepseekKey) {
      throw new Error('DeepSeek API key not configured.');
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
    if (!this.adalineGateway) {
      throw new Error('Adaline Gateway not initialized. Please check your API key configuration.');
    }

    try {
      // Build message history for Adaline
      const messages = this.buildChatMessages(message, history);

      // TODO: Fix Gateway interface - temporarily disabled
      // Use Adaline Gateway for completion with OpenAI as default provider
      // const response = await this.adalineGateway.openai.chat.completions.create({
      //   messages,
      //   model: 'gpt-4-turbo',
      //   temperature: 0.7,
      //   maxTokens: 4096
      // });
      const response = { choices: [{ message: { content: 'OpenAI response via Adaline (placeholder)' } }] };

      const responseText = response.choices[0]?.message?.content || '';

      return {
        text: responseText,
        provider: 'adaline',
        metadata: {
          model: 'gpt-4-turbo',
          usage: { prompt_tokens: 0, completion_tokens: 0, total_tokens: 0 },
          gateway: 'adaline'
        },
      };
    } catch (error) {
      console.error('Adaline chat error:', error);
      throw new Error(`Adaline error: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  private buildChatMessages(
    message: string,
    history?: ChatMessage[]
  ): Array<{ role: string; content: string }> {
    const messages: Array<{ role: string; content: string }> = [];

    // Add history if present
    if (history && history.length > 0) {
      history.forEach((msg) => {
        messages.push({
          role: msg.type === 'user' ? 'user' : 'assistant',
          content: msg.text,
        });
      });
    }

    // Add current message
    messages.push({
      role: 'user',
      content: message,
    });

    return messages;
  }

  // Provider availability checks
  isProviderAvailable(provider: LLMProvider): boolean {
    switch (provider) {
      case 'gemini':
        return this.geminiModel !== null;
      case 'openai':
        return this.adalineGateway !== null && !!import.meta.env.VITE_ADALINE_API_KEY;
      case 'adaline':
        return this.adalineGateway !== null && !!import.meta.env.VITE_ADALINE_API_KEY;
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
}

// Singleton instance
let llmProviderServiceInstance: LLMProviderService | null = null;

export const getLLMProviderService = (): LLMProviderService => {
  if (!llmProviderServiceInstance) {
    llmProviderServiceInstance = new LLMProviderService();
  }
  return llmProviderServiceInstance;
};

export default getLLMProviderService;