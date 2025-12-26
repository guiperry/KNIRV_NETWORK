// LLM Provider Service - Multi-LLM Support
import { Adaline } from '@adaline/gateway';
import { GoogleGenerativeAI, GenerativeModel } from '@google/generative-ai';
import type { LLMProvider, ChatMessage, ChatResponse } from '../types/chatBrain';

export class LLMProviderService {
  private adaline: Adaline | null = null;
  private gemini: GoogleGenerativeAI | null = null;
  private geminiModel: GenerativeModel | null = null;

  constructor() {
    this.initializeProviders();
  }

  private initializeProviders(): void {
    // Initialize Adaline (for OpenAI and multi-provider support)
    const adalineKey = import.meta.env.VITE_ADALINE_KEY;
    if (adalineKey) {
      try {
        this.adaline = new Adaline({ apiKey: adalineKey });
      } catch (error) {
        console.error('Failed to initialize Adaline:', error);
      }
    }

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

  private buildGeminiHistory(history?: ChatMessage[]): Array<{ role: string; parts: string }> {
    if (!history || history.length === 0) return [];

    return history.map((msg) => ({
      role: msg.type === 'user' ? 'user' : 'model',
      parts: msg.text,
    }));
  }

  private async openAIChat(message: string, history?: ChatMessage[]): Promise<ChatResponse> {
    if (!this.adaline) {
      throw new Error('Adaline not initialized. Please check your API key.');
    }

    try {
      const messages = this.buildChatMessages(message, history);

      const response = await this.adaline.chat({
        messages,
        model: 'gpt-4-turbo-preview', // or gpt-3.5-turbo
      });

      return {
        text: response.content || '',
        provider: 'openai',
        metadata: {
          model: 'gpt-4-turbo-preview',
          usage: response.usage,
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
    if (!this.adaline) {
      throw new Error('Adaline not initialized. Please check your API key.');
    }

    try {
      const messages = this.buildChatMessages(message, history);

      // Let Adaline auto-select the best model
      const response = await this.adaline.chat({
        messages,
        // No model specified - Adaline will choose automatically
      });

      return {
        text: response.content || '',
        provider: 'adaline',
        metadata: {
          model: response.model || 'auto',
          usage: response.usage,
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
      case 'adaline':
        return this.adaline !== null;
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
