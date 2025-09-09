import React, { useState, useEffect } from 'react';
import { Brain, MessageSquare, Send, User, Bot, Settings, Zap, Database } from 'lucide-react';
import { useAuth } from '../AuthContext';

interface ChatMessage {
  id: string;
  content: string;
  sender: 'user' | 'brain';
  timestamp: Date;
  confidence?: number;
  memoryUsed?: string[];
}

interface BrainMemory {
  id: string;
  type: 'conversation' | 'fact' | 'preference' | 'skill';
  content: string;
  timestamp: Date;
  relevance: number;
}

const MyChatBrain: React.FC = () => {
  const { user } = useAuth();
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [inputMessage, setInputMessage] = useState('');
  const [isProcessing, setIsProcessing] = useState(false);
  const [brainMemories, setBrainMemories] = useState<BrainMemory[]>([]);
  const [showMemoryPanel, setShowMemoryPanel] = useState(false);

  useEffect(() => {
    // Initialize with welcome message and sample memories
    setMessages([
      {
        id: '1',
        content: `Hello ${user?.username || 'there'}! I'm your personal ChatBrain. I remember our conversations and learn from your preferences to provide better assistance over time.`,
        sender: 'brain',
        timestamp: new Date(),
        confidence: 0.95
      }
    ]);

    setBrainMemories([
      {
        id: '1',
        type: 'preference',
        content: 'User prefers detailed technical explanations',
        timestamp: new Date(Date.now() - 86400000),
        relevance: 0.8
      },
      {
        id: '2',
        type: 'fact',
        content: 'User is working on KNIRV network development',
        timestamp: new Date(Date.now() - 172800000),
        relevance: 0.9
      },
      {
        id: '3',
        type: 'skill',
        content: 'User has experience with TypeScript and React',
        timestamp: new Date(Date.now() - 259200000),
        relevance: 0.7
      }
    ]);
  }, [user]);

  const handleSendMessage = async () => {
    if (!inputMessage.trim() || isProcessing) return;

    const userMessage: ChatMessage = {
      id: Date.now().toString(),
      content: inputMessage,
      sender: 'user',
      timestamp: new Date()
    };

    setMessages(prev => [...prev, userMessage]);
    setInputMessage('');
    setIsProcessing(true);

    // Simulate brain processing with memory retrieval
    try {
      await new Promise(resolve => setTimeout(resolve, 1500));

      // Simulate memory retrieval
      const relevantMemories = brainMemories
        .filter(memory => memory.relevance > 0.6)
        .map(memory => memory.content);

      const brainResponse: ChatMessage = {
        id: (Date.now() + 1).toString(),
        content: `Based on what I remember about you and our previous conversations, here's my response: [AI response would be generated here using personal context and memories]`,
        sender: 'brain',
        timestamp: new Date(),
        confidence: 0.87,
        memoryUsed: relevantMemories.slice(0, 2)
      };

      setMessages(prev => [...prev, brainResponse]);

      // Add new memory from this conversation
      const newMemory: BrainMemory = {
        id: Date.now().toString(),
        type: 'conversation',
        content: `User asked: "${inputMessage.substring(0, 50)}${inputMessage.length > 50 ? '...' : ''}"`,
        timestamp: new Date(),
        relevance: 0.6
      };

      setBrainMemories(prev => [newMemory, ...prev].slice(0, 10)); // Keep only recent memories
    } catch (error) {
      console.error('Brain processing error:', error);
    } finally {
      setIsProcessing(false);
    }
  };

  const getMemoryTypeIcon = (type: string) => {
    switch (type) {
      case 'conversation':
        return <MessageSquare className="w-4 h-4 text-blue-400" />;
      case 'fact':
        return <Database className="w-4 h-4 text-green-400" />;
      case 'preference':
        return <Settings className="w-4 h-4 text-purple-400" />;
      case 'skill':
        return <Zap className="w-4 h-4 text-yellow-400" />;
      default:
        return <Brain className="w-4 h-4 text-slate-400" />;
    }
  };

  return (
    <div className="flex h-full bg-slate-900">
      {/* Main Chat Area */}
      <div className="flex-1 flex flex-col">
        {/* Header */}
        <div className="bg-slate-800/50 border-b border-slate-700/50 p-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3">
              <div className="p-2 bg-blue-500/20 rounded-lg">
                <Brain className="w-6 h-6 text-blue-400" />
              </div>
              <div>
                <h1 className="text-xl font-bold text-white">MyChatBrain</h1>
                <p className="text-sm text-slate-400">Personal AI with memory</p>
              </div>
            </div>
            <button
              onClick={() => setShowMemoryPanel(!showMemoryPanel)}
              className="px-3 py-2 bg-slate-700/50 hover:bg-slate-600/50 text-white rounded-lg transition-colors duration-200"
            >
              Memory
            </button>
          </div>
        </div>

        {/* Messages */}
        <div className="flex-1 overflow-y-auto p-4 space-y-4">
          {messages.map((message) => (
            <div
              key={message.id}
              className={`flex ${message.sender === 'user' ? 'justify-end' : 'justify-start'}`}
            >
              <div
                className={`max-w-xs lg:max-w-md px-4 py-2 rounded-lg ${
                  message.sender === 'user'
                    ? 'bg-blue-600 text-white'
                    : 'bg-slate-700 text-white'
                }`}
              >
                <div className="flex items-center space-x-2 mb-1">
                  {message.sender === 'user' ? (
                    <User className="w-4 h-4" />
                  ) : (
                    <Brain className="w-4 h-4" />
                  )}
                  <span className="text-xs opacity-75">
                    {message.timestamp.toLocaleTimeString()}
                  </span>
                  {message.confidence && (
                    <span className="text-xs bg-green-500/20 text-green-300 px-2 py-1 rounded">
                      {Math.round(message.confidence * 100)}% confident
                    </span>
                  )}
                </div>
                <p className="text-sm">{message.content}</p>
                {message.memoryUsed && message.memoryUsed.length > 0 && (
                  <div className="mt-2 text-xs text-slate-300">
                    <div className="font-medium">Used memories:</div>
                    {message.memoryUsed.map((memory, index) => (
                      <div key={index} className="text-slate-400">• {memory}</div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>

        {/* Input */}
        <div className="bg-slate-800/50 border-t border-slate-700/50 p-4">
          <div className="flex space-x-2">
            <input
              type="text"
              value={inputMessage}
              onChange={(e) => setInputMessage(e.target.value)}
              onKeyPress={(e) => e.key === 'Enter' && handleSendMessage()}
              placeholder="Chat with your personal AI brain..."
              className="flex-1 bg-slate-700/50 border border-slate-600/50 rounded-lg px-4 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500/50"
              disabled={isProcessing}
            />
            <button
              onClick={handleSendMessage}
              disabled={isProcessing || !inputMessage.trim()}
              className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-slate-600 disabled:cursor-not-allowed text-white rounded-lg transition-colors duration-200"
            >
              <Send className="w-5 h-5" />
            </button>
          </div>
        </div>
      </div>

      {/* Memory Panel */}
      {showMemoryPanel && (
        <div className="w-80 bg-slate-800/50 border-l border-slate-700/50 p-4">
          <h3 className="text-lg font-bold text-white mb-4">Brain Memory</h3>
          <div className="space-y-3">
            {brainMemories.map((memory) => (
              <div key={memory.id} className="bg-slate-700/30 rounded-lg p-3">
                <div className="flex items-center space-x-2 mb-2">
                  {getMemoryTypeIcon(memory.type)}
                  <span className="text-sm font-medium text-white capitalize">{memory.type}</span>
                  <span className="text-xs text-slate-400 ml-auto">
                    {Math.round(memory.relevance * 100)}%
                  </span>
                </div>
                <p className="text-sm text-slate-300">{memory.content}</p>
                <p className="text-xs text-slate-500 mt-1">
                  {memory.timestamp.toLocaleDateString()}
                </p>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

export default MyChatBrain;
