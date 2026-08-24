import React, { useState, useEffect, useRef } from 'react';
import { MessageSquare, Send, Bot, User, Brain, Zap } from 'lucide-react';
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from './AuthContext';
import ChatChain from './chat/ChatChain';
import MyChatBrain from './chat/MyChatBrain';

interface Message {
  id: string;
  content: string;
  sender: 'user' | 'assistant' | 'system';
  timestamp: Date;
  type?: 'text' | 'code' | 'error';
}

interface ChatSession {
  id: string;
  name: string;
  messages: Message[];
  lastActivity: Date;
}

export const Chat: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { canAccessSubPage } = useAuth();
  const [sessions, setSessions] = useState<ChatSession[]>([]);
  const [activeSession, setActiveSession] = useState<string | null>(null);
  const [inputMessage, setInputMessage] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Check if we're on a sub-route
  const isSubRoute = location.pathname !== '/chat';

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [sessions]);

  const createNewSession = () => {
    const newSession: ChatSession = {
      id: `session-${Date.now()}`,
      name: `Chat ${sessions.length + 1}`,
      messages: [],
      lastActivity: new Date()
    };
    setSessions([...sessions, newSession]);
    setActiveSession(newSession.id);
  };

  const sendMessage = async () => {
    if (!inputMessage.trim() || !activeSession) return;

    const userMessage: Message = {
      id: `msg-${Date.now()}`,
      content: inputMessage,
      sender: 'user',
      timestamp: new Date()
    };

    // Update session with user message
    setSessions(prev => prev.map(session => 
      session.id === activeSession 
        ? { ...session, messages: [...session.messages, userMessage], lastActivity: new Date() }
        : session
    ));

    setInputMessage('');
    setIsLoading(true);

    // Simulate AI response
    setTimeout(() => {
      const assistantMessage: Message = {
        id: `msg-${Date.now()}-assistant`,
        content: `I understand you said: "${userMessage.content}". How can I help you with the KNIRV Network?`,
        sender: 'assistant',
        timestamp: new Date()
      };

      setSessions(prev => prev.map(session => 
        session.id === activeSession 
          ? { ...session, messages: [...session.messages, assistantMessage] }
          : session
      ));
      setIsLoading(false);
    }, 1000);
  };

  const currentSession = sessions.find(s => s.id === activeSession);

  // If we're on a sub-route, render the sub-component
  if (isSubRoute) {
    return (
      <Routes>
        <Route path="/chatchain" element={<ChatChain />} />
        <Route path="/mychatbrain" element={<MyChatBrain />} />
      </Routes>
    );
  }

  return (
    <div className="flex h-full bg-slate-900">
      {/* Chat Navigation Sidebar */}
      <div className="w-80 bg-slate-800/50 border-r border-slate-700/50 flex flex-col">
        <div className="p-4 border-b border-slate-700/50">
          <h2 className="text-lg font-bold text-white mb-4">Chat Options</h2>
          <div className="space-y-2">
            {canAccessSubPage('chat', 'chatchain') && (
              <button
                onClick={() => navigate('/chat/chatchain')}
                className="w-full text-left p-3 rounded-lg bg-purple-600/20 hover:bg-purple-600/30 border border-purple-500/30 transition-colors flex items-center gap-3"
              >
                <Zap className="w-5 h-5 text-purple-400" />
                <div>
                  <div className="text-white font-medium">ChatChain</div>
                  <div className="text-sm text-purple-200">Multi-step AI task execution</div>
                </div>
              </button>
            )}
            {canAccessSubPage('chat', 'mychatbrain') && (
              <button
                onClick={() => navigate('/chat/mychatbrain')}
                className="w-full text-left p-3 rounded-lg bg-blue-600/20 hover:bg-blue-600/30 border border-blue-500/30 transition-colors flex items-center gap-3"
              >
                <Brain className="w-5 h-5 text-blue-400" />
                <div>
                  <div className="text-white font-medium">MyChatBrain</div>
                  <div className="text-sm text-blue-200">Personal AI with memory</div>
                </div>
              </button>
            )}
          </div>
        </div>

        <div className="p-4 border-b border-slate-700/50">
          <button
            onClick={createNewSession}
            className="w-full bg-slate-600 text-white px-4 py-2 rounded-lg hover:bg-slate-500 transition-colors flex items-center justify-center gap-2"
          >
            <MessageSquare className="w-4 h-4" />
            New Chat Session
          </button>
        </div>
        
        <div className="flex-1 overflow-y-auto">
          {sessions.map(session => (
            <div
              key={session.id}
              onClick={() => setActiveSession(session.id)}
              className={`p-4 border-b border-slate-700/30 cursor-pointer hover:bg-slate-700/30 transition-colors ${
                activeSession === session.id ? 'bg-slate-600/50 border-l-4 border-l-blue-500' : ''
              }`}
            >
              <div className="font-medium text-white">{session.name}</div>
              <div className="text-sm text-slate-400 mt-1">
                {session.messages.length} messages
              </div>
              <div className="text-xs text-slate-500 mt-1">
                {session.lastActivity.toLocaleTimeString()}
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Chat Interface */}
      <div className="flex-1 flex flex-col">
        {activeSession ? (
          <>
            {/* Chat Header */}
            <div className="bg-slate-800/50 border-b border-slate-700/50 p-4">
              <h2 className="text-lg font-semibold text-white">
                {currentSession?.name}
              </h2>
              <p className="text-sm text-slate-400">
                KNIRV Network AI Assistant
              </p>
            </div>

            {/* Messages */}
            <div className="flex-1 overflow-y-auto p-4 space-y-4">
              {currentSession?.messages.map(message => (
                <div
                  key={message.id}
                  className={`flex ${message.sender === 'user' ? 'justify-end' : 'justify-start'}`}
                >
                  <div className={`max-w-xs lg:max-w-md px-4 py-2 rounded-lg ${
                    message.sender === 'user'
                      ? 'bg-blue-600 text-white'
                      : 'bg-slate-700 border border-slate-600 text-white'
                  }`}>
                    <div className="flex items-center gap-2 mb-1">
                      {message.sender === 'user' ? (
                        <User className="w-4 h-4" />
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

              {isLoading && (
                <div className="flex justify-start">
                  <div className="bg-slate-700 border border-slate-600 text-white max-w-xs lg:max-w-md px-4 py-2 rounded-lg">
                    <div className="flex items-center gap-2">
                      <Bot className="w-4 h-4" />
                      <div className="flex space-x-1">
                        <div className="w-2 h-2 bg-slate-400 rounded-full animate-bounce"></div>
                        <div className="w-2 h-2 bg-slate-400 rounded-full animate-bounce" style={{ animationDelay: '0.1s' }}></div>
                        <div className="w-2 h-2 bg-slate-400 rounded-full animate-bounce" style={{ animationDelay: '0.2s' }}></div>
                      </div>
                    </div>
                  </div>
                </div>
              )}
              <div ref={messagesEndRef} />
            </div>

            {/* Input */}
            <div className="bg-slate-800/50 border-t border-slate-700/50 p-4">
              <div className="flex gap-2">
                <input
                  type="text"
                  value={inputMessage}
                  onChange={(e) => setInputMessage(e.target.value)}
                  onKeyPress={(e) => e.key === 'Enter' && sendMessage()}
                  placeholder="Type your message..."
                  className="flex-1 bg-slate-700/50 border border-slate-600/50 rounded-lg px-4 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500/50"
                  disabled={isLoading}
                />
                <button
                  onClick={sendMessage}
                  disabled={!inputMessage.trim() || isLoading}
                  className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                >
                  <Send className="w-4 h-4" />
                </button>
              </div>
            </div>
          </>
        ) : (
          <div className="flex-1 flex items-center justify-center">
            <div className="text-center">
              <MessageSquare className="w-16 h-16 text-slate-400 mx-auto mb-4" />
              <h3 className="text-lg font-medium text-white mb-2">
                Welcome to KNIRV Chat
              </h3>
              <p className="text-slate-400 mb-6">
                Choose a chat mode or create a new session to get started
              </p>
              <div className="space-y-3">
                {canAccessSubPage('chat', 'chatchain') && (
                  <button
                    onClick={() => navigate('/chat/chatchain')}
                    className="block w-full bg-purple-600 text-white px-6 py-3 rounded-lg hover:bg-purple-700 transition-colors"
                  >
                    Try ChatChain
                  </button>
                )}
                {canAccessSubPage('chat', 'mychatbrain') && (
                  <button
                    onClick={() => navigate('/chat/mychatbrain')}
                    className="block w-full bg-blue-600 text-white px-6 py-3 rounded-lg hover:bg-blue-700 transition-colors"
                  >
                    Try MyChatBrain
                  </button>
                )}
                <button
                  onClick={createNewSession}
                  className="block w-full bg-slate-600 text-white px-6 py-3 rounded-lg hover:bg-slate-500 transition-colors"
                >
                  Start Regular Chat
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
