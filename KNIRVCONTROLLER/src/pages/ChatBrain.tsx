// Chat-Brain Page - Main Orchestrator Component
import React, { useState } from 'react';
import { Network, FileText, Trash2 } from 'lucide-react';
import { ChatBrainProvider } from '../contexts/ChatBrainContext';
import { ChatInterface } from '../components/chat-brain/ChatInterface';
import { MemoryGraphView } from '../components/chat-brain/MemoryGraphView';
import { NotesPanel } from '../components/chat-brain/NotesPanel';
import { LLMSelector } from '../components/chat-brain/LLMSelector';
import { useChatBrain } from '../contexts/ChatBrainContext';

function ChatBrainContent() {
  const { clearChat } = useChatBrain();
  const [showMemoryGraph, setShowMemoryGraph] = useState(false);
  const [showNotes, setShowNotes] = useState(false);

  const handleClearChat = () => {
    if (confirm('Are you sure you want to clear the chat history?')) {
      clearChat();
    }
  };

  return (
    <div className="min-h-screen bg-gray-900 text-white flex flex-col">
      {/* Header */}
      <div className="bg-gray-800/80 backdrop-blur-sm border-b border-gray-700/50 p-4">
        <div className="max-w-7xl mx-auto flex items-center justify-between">
          {/* Left: Title and LLM Selector */}
          <div className="flex items-center space-x-4">
            <h1 className="text-2xl font-bold text-white flex items-center space-x-2">
              <span>🧠</span>
              <span>Chat-Brain</span>
            </h1>
            <LLMSelector />
          </div>

          {/* Right: Action Buttons */}
          <div className="flex items-center space-x-2">
            <button
              onClick={() => setShowMemoryGraph(!showMemoryGraph)}
              className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors flex items-center space-x-2 ${
                showMemoryGraph
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-700 hover:bg-gray-600 text-gray-300'
              }`}
            >
              <Network size={16} />
              <span>Memory Graph</span>
            </button>

            <button
              onClick={() => setShowNotes(!showNotes)}
              className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors flex items-center space-x-2 ${
                showNotes
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-700 hover:bg-gray-600 text-gray-300'
              }`}
            >
              <FileText size={16} />
              <span>Notes</span>
            </button>

            <button
              onClick={handleClearChat}
              className="px-4 py-2 bg-gray-700 hover:bg-gray-600 text-gray-300 rounded-lg text-sm font-medium transition-colors flex items-center space-x-2"
            >
              <Trash2 size={16} />
              <span>Clear Chat</span>
            </button>
          </div>
        </div>
      </div>

      {/* Main Chat Interface */}
      <ChatInterface />

      {/* Side Panels */}
      {showMemoryGraph && <MemoryGraphView onClose={() => setShowMemoryGraph(false)} />}
      {showNotes && <NotesPanel onClose={() => setShowNotes(false)} />}
    </div>
  );
}

export default function ChatBrain() {
  return (
    <ChatBrainProvider>
      <ChatBrainContent />
    </ChatBrainProvider>
  );
}
