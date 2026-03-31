// LLM Provider Selector Component
import React from 'react';
import { useChatBrain } from '../../contexts/ChatBrainContext';
import { getChatBrainService } from '../../services/chatBrainService';

export function LLMSelector() {
  const { selectedProvider, setSelectedProvider, availableProviders } = useChatBrain();
  const chatBrainService = getChatBrainService();

  const providers = availableProviders.map((provider) => ({
    id: provider,
    name: chatBrainService.getProviderName(provider),
    icon: chatBrainService.getProviderIcon(provider),
  }));

  // If no providers available, show warning
  if (providers.length === 0) {
    return (
      <div className="text-sm text-red-400 flex items-center space-x-2">
        <span>⚠️</span>
        <span>No LLM providers configured</span>
      </div>
    );
  }

  return (
    <div className="flex items-center space-x-2">
      <label htmlFor="llm-selector" className="text-sm text-gray-400">
        Model:
      </label>
      <select
        id="llm-selector"
        value={selectedProvider}
        onChange={(e) => setSelectedProvider(e.target.value as any)}
        className="bg-gray-800 text-white border border-gray-700 rounded-lg px-3 py-1.5 text-sm
                   focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent
                   hover:bg-gray-700 transition-colors cursor-pointer"
      >
        {providers.map((provider) => (
          <option key={provider.id} value={provider.id}>
            {provider.icon} {provider.name}
          </option>
        ))}
      </select>
    </div>
  );
}
