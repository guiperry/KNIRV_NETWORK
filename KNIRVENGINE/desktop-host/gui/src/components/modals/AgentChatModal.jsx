import React, { useState, useEffect, useRef } from 'react';
import {
  X,
  Send,
  MessageSquare,
  Bot,
  User,
  Loader2,
  Terminal,
  Activity,
  AlertCircle
} from 'lucide-react';
import { handleApiError } from '../../utils/errorHandler';

const AgentChatModal = ({ isOpen, onClose, agent }) => {
  const [messages, setMessages] = useState([]);
  const [inputMessage, setInputMessage] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [activeTab, setActiveTab] = useState('chat');
  const [agentLogs, setAgentLogs] = useState([]);
  const messagesEndRef = useRef(null);
  const logsEndRef = useRef(null);

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    if (activeTab === 'chat') {
      messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    } else {
      logsEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }
  }, [messages, agentLogs, activeTab]);

  // Load initial chat history and logs when modal opens
  useEffect(() => {
    if (isOpen && agent) {
      loadChatHistory();
      loadAgentLogs();
    }
  }, [isOpen, agent]);

  const loadChatHistory = async () => {
    try {
      // Fetch chat history from the API
      const response = await fetch(`/api/v1/agents/${agent.id}/chat/history`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      });

      if (response.ok) {
        const data = await response.json();
        if (data.status === 'success' && data.history) {
          setMessages(data.history);
        } else {
          // Fallback to welcome message
          setMessages([
            {
              id: 1,
              type: 'agent',
              content: `Hello! I'm ${agent.name}. How can I assist you today?`,
              timestamp: new Date().toISOString()
            }
          ]);
        }
      } else {
        // Fallback to welcome message on error
        setMessages([
          {
            id: 1,
            type: 'agent',
            content: `Hello! I'm ${agent.name}. How can I assist you today?`,
            timestamp: new Date().toISOString()
          }
        ]);
      }
    } catch (error) {
      console.error('Error loading chat history:', error);
      handleApiError(error, {
        operation: 'load_chat_history',
        component: 'AgentChatModal',
        agentId: agent.id,
        timestamp: new Date().toISOString(),
        context: 'Loading chat history for agent interaction'
      });

      // Fallback to welcome message
      setMessages([
        {
          id: 1,
          type: 'agent',
          content: `Hello! I'm ${agent.name}. How can I assist you today?`,
          timestamp: new Date().toISOString()
        }
      ]);
    }
  };

  const loadAgentLogs = async () => {
    try {
      // Try to fetch agent logs from the API
      const response = await fetch(`/api/v1/agents/${agent.id}/sub-agents/${agent.id}/logs`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      });

      if (response.ok) {
        const data = await response.json();
        if (data.status === 'success' && data.logs) {
          setAgentLogs(data.logs);
        } else {
          // Fallback to simulated logs
          setAgentLogs([
            {
              id: 1,
              level: 'info',
              message: `Agent ${agent.name} initialized successfully`,
              timestamp: new Date(Date.now() - 60000).toISOString()
            },
            {
              id: 2,
              level: 'info',
              message: 'Waiting for user interaction...',
              timestamp: new Date(Date.now() - 30000).toISOString()
            },
            {
              id: 3,
              level: 'info',
              message: 'Chat session started',
              timestamp: new Date().toISOString()
            }
          ]);
        }
      } else {
        // Fallback to simulated logs on error
        setAgentLogs([
          {
            id: 1,
            level: 'info',
            message: `Agent ${agent.name} initialized successfully`,
            timestamp: new Date(Date.now() - 60000).toISOString()
          },
          {
            id: 2,
            level: 'info',
            message: 'Waiting for user interaction...',
            timestamp: new Date(Date.now() - 30000).toISOString()
          },
          {
            id: 3,
            level: 'info',
            message: 'Chat session started',
            timestamp: new Date().toISOString()
          }
        ]);
      }
    } catch (error) {
      console.error('Error loading agent logs:', error);
      handleApiError(error, {
        operation: 'load_agent_logs',
        component: 'AgentChatModal',
        agentId: agent.id,
        timestamp: new Date().toISOString(),
        context: 'Loading agent logs for monitoring'
      });

      // Fallback to simulated logs
      setAgentLogs([
        {
          id: 1,
          level: 'info',
          message: `Agent ${agent.name} initialized successfully`,
          timestamp: new Date(Date.now() - 60000).toISOString()
        },
        {
          id: 2,
          level: 'info',
          message: 'Waiting for user interaction...',
          timestamp: new Date(Date.now() - 30000).toISOString()
        },
        {
          id: 3,
          level: 'info',
          message: 'Chat session started',
          timestamp: new Date().toISOString()
        }
      ]);
    }
  };

  const handleSendMessage = async (e) => {
    e.preventDefault();
    
    if (!inputMessage.trim() || isLoading) return;

    const userMessage = {
      id: Date.now(),
      type: 'user',
      content: inputMessage.trim(),
      timestamp: new Date().toISOString()
    };

    setMessages(prev => [...prev, userMessage]);
    setInputMessage('');
    setIsLoading(true);

    try {
      // Determine if this is a WASM agent
      const isWASMAgent = agent.type === 'wasm' ||
                         (agent.config && JSON.parse(agent.config).build_target === 'wasm') ||
                         (agent.plugin_info && agent.plugin_info.filename && agent.plugin_info.filename.endsWith('.wasm'));

      // Use appropriate endpoint based on agent type
      const endpoint = isWASMAgent ? '/api/v1/adk/agents/message' : '/api/v1/agents/chat';

      const requestBody = isWASMAgent ? {
        agentId: agent.id,
        message: userMessage.content,
        type: 'user_input'
      } : {
        agentId: agent.id,
        message: userMessage.content,
        type: 'user_input',
        sessionId: `chat-${agent.id}-${Date.now()}`
      };

      // Send message to the agent API
      const response = await fetch(endpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(requestBody),
      });

      if (response.ok) {
        const data = await response.json();

        if (data.status === 'success' && data.response) {
          const agentResponse = {
            id: Date.now() + 1,
            type: 'agent',
            content: data.response,
            timestamp: new Date().toISOString()
          };

          setMessages(prev => [...prev, agentResponse]);

          // Add log entry
          setAgentLogs(prev => [...prev, {
            id: Date.now() + 2,
            level: 'info',
            message: `Processed user message and generated response`,
            timestamp: new Date().toISOString()
          }]);
        } else {
          // Handle API error response
          const errorMessage = {
            id: Date.now() + 1,
            type: 'error',
            content: data.message || 'Sorry, I encountered an error processing your message.',
            timestamp: new Date().toISOString()
          };

          setMessages(prev => [...prev, errorMessage]);
        }
      } else {
        // Handle HTTP error
        const errorMessage = {
          id: Date.now() + 1,
          type: 'error',
          content: 'Sorry, I encountered a network error. Please try again.',
          timestamp: new Date().toISOString()
        };

        setMessages(prev => [...prev, errorMessage]);
      }

    } catch (error) {
      console.error('Error sending message:', error);
      handleApiError(error, {
        operation: 'send_message',
        component: 'AgentChatModal',
        agentId: agent.id,
        message: userMessage.content,
        timestamp: new Date().toISOString(),
        context: 'User attempted to send message to agent'
      });

      const errorMessage = {
        id: Date.now() + 1,
        type: 'error',
        content: 'Sorry, I encountered an error processing your message. Please try again.',
        timestamp: new Date().toISOString()
      };

      setMessages(prev => [...prev, errorMessage]);
    } finally {
      setIsLoading(false);
    }
  };

  const formatTimestamp = (timestamp) => {
    return new Date(timestamp).toLocaleTimeString();
  };

  const getLogLevelColor = (level) => {
    switch (level) {
      case 'error': return 'text-red-400';
      case 'warn': return 'text-yellow-400';
      case 'info': return 'text-blue-400';
      case 'debug': return 'text-slate-400';
      default: return 'text-slate-300';
    }
  };

  if (!isOpen || !agent) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div 
        className="absolute inset-0 bg-black bg-opacity-70 backdrop-blur-sm"
        onClick={onClose}
      ></div>
      
      {/* Modal */}
      <div className="relative bg-slate-800 rounded-xl border border-slate-700 w-full max-w-2xl max-h-[80vh] overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-slate-700">
          <div className="flex items-center space-x-3">
            <div className="p-2 bg-gradient-to-r from-purple-500/20 to-blue-500/20 rounded-lg border border-purple-500/30">
              <MessageSquare className="w-5 h-5 text-purple-400" />
            </div>
            <div>
              <h2 className="text-lg font-bold text-white">Chat with {agent.name}</h2>
              <p className="text-slate-400 text-sm">Status: {agent.status}</p>
            </div>
          </div>
          <button
            onClick={onClose}
            aria-label="Close"
            className="p-2 text-slate-400 hover:text-white transition-colors duration-200"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Tabs */}
        <div className="flex border-b border-slate-700">
          <button
            onClick={() => setActiveTab('chat')}
            className={`flex-1 px-4 py-3 text-sm font-medium transition-colors ${
              activeTab === 'chat'
                ? 'text-purple-400 border-b-2 border-purple-400 bg-slate-700/30'
                : 'text-slate-400 hover:text-white'
            }`}
          >
            <MessageSquare className="w-4 h-4 inline mr-2" />
            Chat
          </button>
          <button
            onClick={() => setActiveTab('logs')}
            className={`flex-1 px-4 py-3 text-sm font-medium transition-colors ${
              activeTab === 'logs'
                ? 'text-purple-400 border-b-2 border-purple-400 bg-slate-700/30'
                : 'text-slate-400 hover:text-white'
            }`}
          >
            <Activity className="w-4 h-4 inline mr-2" />
            Activity Logs
          </button>
        </div>

        {/* Content */}
        <div className="flex flex-col h-96">
          {activeTab === 'chat' ? (
            <>
              {/* Messages */}
              <div className="flex-1 overflow-y-auto p-4 space-y-4">
                {messages.map((message) => (
                  <div
                    key={message.id}
                    className={`flex ${message.type === 'user' ? 'justify-end' : 'justify-start'}`}
                  >
                    <div
                      className={`max-w-xs lg:max-w-md px-4 py-2 rounded-lg ${
                        message.type === 'user'
                          ? 'bg-purple-600 text-white'
                          : message.type === 'error'
                          ? 'bg-red-600/20 border border-red-600/30 text-red-400'
                          : 'bg-slate-700 text-slate-100'
                      }`}
                    >
                      <div className="flex items-start space-x-2">
                        {message.type === 'agent' && <Bot className="w-4 h-4 mt-0.5 text-blue-400" />}
                        {message.type === 'user' && <User className="w-4 h-4 mt-0.5" />}
                        {message.type === 'error' && <AlertCircle className="w-4 h-4 mt-0.5" />}
                        <div className="flex-1">
                          <p className="text-sm">{message.content}</p>
                          <p className="text-xs opacity-70 mt-1">{formatTimestamp(message.timestamp)}</p>
                        </div>
                      </div>
                    </div>
                  </div>
                ))}
                {isLoading && (
                  <div className="flex justify-start">
                    <div className="bg-slate-700 text-slate-100 px-4 py-2 rounded-lg">
                      <div className="flex items-center space-x-2">
                        <Bot className="w-4 h-4 text-blue-400" />
                        <Loader2 className="w-4 h-4 animate-spin" />
                        <span className="text-sm">Thinking...</span>
                      </div>
                    </div>
                  </div>
                )}
                <div ref={messagesEndRef} />
              </div>

              {/* Input */}
              <div className="border-t border-slate-700 p-4">
                <form onSubmit={handleSendMessage} className="flex space-x-2">
                  <input
                    type="text"
                    value={inputMessage}
                    onChange={(e) => setInputMessage(e.target.value)}
                    placeholder="Type your message..."
                    className="flex-1 bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500"
                    disabled={isLoading}
                  />
                  <button
                    type="submit"
                    disabled={!inputMessage.trim() || isLoading}
                    className="px-4 py-2 bg-purple-600 hover:bg-purple-700 disabled:bg-slate-600 disabled:cursor-not-allowed text-white rounded-lg transition-colors"
                  >
                    <Send className="w-4 h-4" />
                  </button>
                </form>
              </div>
            </>
          ) : (
            /* Activity Logs */
            <div className="flex-1 overflow-y-auto p-4">
              <div className="space-y-2">
                {agentLogs.map((log) => (
                  <div key={log.id} className="flex items-start space-x-3 text-sm">
                    <span className="text-slate-500 text-xs w-16 flex-shrink-0">
                      {formatTimestamp(log.timestamp)}
                    </span>
                    <span className={`uppercase text-xs font-medium w-12 flex-shrink-0 ${getLogLevelColor(log.level)}`}>
                      {log.level}
                    </span>
                    <span className="text-slate-300 flex-1">{log.message}</span>
                  </div>
                ))}
                <div ref={logsEndRef} />
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default AgentChatModal;
