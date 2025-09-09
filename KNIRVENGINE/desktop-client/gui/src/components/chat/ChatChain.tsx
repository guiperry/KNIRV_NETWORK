import React, { useState, useEffect } from 'react';
import { MessageSquare, Send, Bot, User, Zap, Clock, CheckCircle } from 'lucide-react';
import { useAuth } from '../AuthContext';

interface ChatMessage {
  id: string;
  content: string;
  sender: 'user' | 'agent' | 'chain';
  timestamp: Date;
  chainStep?: string;
  status?: 'pending' | 'processing' | 'completed' | 'error';
}

interface ChainStep {
  id: string;
  name: string;
  description: string;
  status: 'pending' | 'processing' | 'completed' | 'error';
  result?: any;
}

const ChatChain: React.FC = () => {
  const { user } = useAuth();
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [inputMessage, setInputMessage] = useState('');
  const [isProcessing, setIsProcessing] = useState(false);
  const [chainSteps, setChainSteps] = useState<ChainStep[]>([]);

  useEffect(() => {
    // Initialize with welcome message
    setMessages([
      {
        id: '1',
        content: 'Welcome to ChatChain! I can help you execute complex multi-step tasks by chaining together different agents and capabilities.',
        sender: 'chain',
        timestamp: new Date(),
        status: 'completed'
      }
    ]);
  }, []);

  const handleSendMessage = async () => {
    if (!inputMessage.trim() || isProcessing) return;

    const userMessage: ChatMessage = {
      id: Date.now().toString(),
      content: inputMessage,
      sender: 'user',
      timestamp: new Date(),
      status: 'completed'
    };

    setMessages(prev => [...prev, userMessage]);
    setInputMessage('');
    setIsProcessing(true);

    // Simulate chain processing
    try {
      const steps: ChainStep[] = [
        { id: '1', name: 'Parse Request', description: 'Understanding your request...', status: 'processing' },
        { id: '2', name: 'Plan Chain', description: 'Creating execution plan...', status: 'pending' },
        { id: '3', name: 'Execute Steps', description: 'Running chain steps...', status: 'pending' },
        { id: '4', name: 'Compile Results', description: 'Gathering results...', status: 'pending' }
      ];

      setChainSteps(steps);

      // Simulate step-by-step processing
      for (let i = 0; i < steps.length; i++) {
        await new Promise(resolve => setTimeout(resolve, 1500));
        
        setChainSteps(prev => prev.map((step, index) => {
          if (index === i) {
            return { ...step, status: 'completed' };
          } else if (index === i + 1) {
            return { ...step, status: 'processing' };
          }
          return step;
        }));
      }

      // Add chain response
      const chainResponse: ChatMessage = {
        id: (Date.now() + 1).toString(),
        content: `I've processed your request through a 4-step chain. Here's what I found: [Chain execution completed successfully]`,
        sender: 'chain',
        timestamp: new Date(),
        status: 'completed'
      };

      setMessages(prev => [...prev, chainResponse]);
    } catch (error) {
      console.error('Chain processing error:', error);
    } finally {
      setIsProcessing(false);
      setChainSteps([]);
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'processing':
        return <div className="animate-spin w-4 h-4 border-2 border-blue-500 border-t-transparent rounded-full" />;
      case 'completed':
        return <CheckCircle className="w-4 h-4 text-green-500" />;
      case 'error':
        return <div className="w-4 h-4 bg-red-500 rounded-full" />;
      default:
        return <Clock className="w-4 h-4 text-slate-400" />;
    }
  };

  return (
    <div className="flex flex-col h-full bg-slate-900">
      {/* Header */}
      <div className="bg-slate-800/50 border-b border-slate-700/50 p-4">
        <div className="flex items-center space-x-3">
          <div className="p-2 bg-purple-500/20 rounded-lg">
            <Zap className="w-6 h-6 text-purple-400" />
          </div>
          <div>
            <h1 className="text-xl font-bold text-white">ChatChain</h1>
            <p className="text-sm text-slate-400">Multi-step AI task execution</p>
          </div>
        </div>
      </div>

      {/* Chain Steps Display */}
      {chainSteps.length > 0 && (
        <div className="bg-slate-800/30 border-b border-slate-700/50 p-4">
          <h3 className="text-sm font-medium text-white mb-3">Chain Execution Progress</h3>
          <div className="space-y-2">
            {chainSteps.map((step) => (
              <div key={step.id} className="flex items-center space-x-3">
                {getStatusIcon(step.status)}
                <div className="flex-1">
                  <div className="text-sm text-white">{step.name}</div>
                  <div className="text-xs text-slate-400">{step.description}</div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

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
                  : message.sender === 'chain'
                  ? 'bg-purple-600/20 text-purple-100 border border-purple-500/30'
                  : 'bg-slate-700 text-white'
              }`}
            >
              <div className="flex items-center space-x-2 mb-1">
                {message.sender === 'user' ? (
                  <User className="w-4 h-4" />
                ) : message.sender === 'chain' ? (
                  <Zap className="w-4 h-4" />
                ) : (
                  <Bot className="w-4 h-4" />
                )}
                <span className="text-xs opacity-75">
                  {message.timestamp.toLocaleTimeString()}
                </span>
              </div>
              <p className="text-sm">{message.content}</p>
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
            placeholder="Describe a complex task for the chain to execute..."
            className="flex-1 bg-slate-700/50 border border-slate-600/50 rounded-lg px-4 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500/50"
            disabled={isProcessing}
          />
          <button
            onClick={handleSendMessage}
            disabled={isProcessing || !inputMessage.trim()}
            className="px-4 py-2 bg-purple-600 hover:bg-purple-700 disabled:bg-slate-600 disabled:cursor-not-allowed text-white rounded-lg transition-colors duration-200"
          >
            <Send className="w-5 h-5" />
          </button>
        </div>
      </div>
    </div>
  );
};

export default ChatChain;
