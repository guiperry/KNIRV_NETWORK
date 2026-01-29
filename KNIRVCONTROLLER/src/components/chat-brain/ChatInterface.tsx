// Chat Interface Component - Main Chat UI
import React, { useState, useRef, useEffect } from 'react';
import { Send, User, Bot, BookmarkPlus } from 'lucide-react';
import { useChatBrain } from '../../contexts/ChatBrainContext';
import ReactMarkdown from 'react-markdown';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';

interface ChatInterfaceProps {
  preloadedPrompt?: string;
}

export function ChatInterface({ preloadedPrompt }: ChatInterfaceProps) {
  const { messages, sendMessage, isLoading, saveNote } = useChatBrain();
  const [input, setInput] = useState('');
  const messagesEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (preloadedPrompt && messages.length === 0) {
      // Auto-send the preloaded prompt after a short delay
      const timer = setTimeout(() => {
        sendMessage(preloadedPrompt);
      }, 500);
      return () => clearTimeout(timer);
    }
  }, [preloadedPrompt, messages.length, sendMessage]);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim() || isLoading) return;

    const messageText = input;
    setInput('');
    await sendMessage(messageText);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit(e);
    }
  };

  const handleSaveAsNote = async (text: string) => {
    try {
      const title = text.split('\n')[0].substring(0, 50) || 'Untitled Note';
      await saveNote(title, text);
      // Could show a toast notification here
    } catch (error) {
      console.error('Failed to save note:', error);
    }
  };

  return (
    <div className="flex-1 flex flex-col bg-gray-900">
      {/* Messages Area */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {messages.length === 0 ? (
          <div className="flex items-center justify-center h-full">
            <div className="text-center text-gray-400 max-w-md">
              <h2 className="text-2xl font-bold mb-2 text-white">Welcome to Chat-Brain</h2>
              <p className="text-gray-500">
                Start a conversation with your AI assistant. Your conversation will be stored in the
                KNIRV network knowledge graph for future reference.
              </p>
            </div>
          </div>
        ) : (
          messages.map((message) => (
            <div
              key={message.id}
              className={`flex items-start space-x-3 ${
                message.type === 'user' ? 'justify-end' : 'justify-start'
              }`}
            >
              {message.type === 'bot' && (
                <div className="w-8 h-8 rounded-full bg-blue-600 flex items-center justify-center flex-shrink-0">
                  <Bot size={20} className="text-white" />
                </div>
              )}

              <div className={`flex-1 max-w-3xl ${message.type === 'user' ? 'order-first' : ''}`}>
                <div
                  className={`rounded-lg p-4 ${
                    message.type === 'user'
                      ? 'bg-blue-600 text-white ml-auto'
                      : message.type === 'system'
                      ? 'bg-red-900/30 border border-red-700 text-red-400'
                      : 'bg-gray-800 text-white'
                  }`}
                >
                  <ReactMarkdown
                    className="prose prose-invert max-w-none"
                    components={{
                      // eslint-disable-next-line @typescript-eslint/no-explicit-any
                      code({ inline, className, children, ...props }: { inline?: boolean; className?: string; children?: React.ReactNode; [key: string]: any }) {
                        const match = /language-(\w+)/.exec(className || '');
                        return !inline && match ? (
                          <SyntaxHighlighter
                            style={vscDarkPlus}
                            language={match[1]}
                            PreTag="div"
                            {...props}
                          >
                            {String(children).replace(/\n$/, '')}
                          </SyntaxHighlighter>
                        ) : (
                          <code className={className} {...props}>
                            {children}
                          </code>
                        );
                      },
                    }}
                  >
                    {message.text}
                  </ReactMarkdown>
                </div>

                {message.type === 'bot' && (
                  <button
                    onClick={() => handleSaveAsNote(message.text)}
                    className="mt-2 text-sm text-gray-400 hover:text-white flex items-center space-x-1 transition-colors"
                  >
                    <BookmarkPlus size={16} />
                    <span>Save as note</span>
                  </button>
                )}
              </div>

              {message.type === 'user' && (
                <div className="w-8 h-8 rounded-full bg-gray-700 flex items-center justify-center flex-shrink-0">
                  <User size={20} className="text-white" />
                </div>
              )}
            </div>
          ))
        )}

        {isLoading && (
          <div className="flex items-start space-x-3">
            <div className="w-8 h-8 rounded-full bg-blue-600 flex items-center justify-center">
              <Bot size={20} className="text-white" />
            </div>
            <div className="bg-gray-800 rounded-lg p-4">
              <div className="flex space-x-2">
                <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce"></div>
                <div
                  className="w-2 h-2 bg-gray-400 rounded-full animate-bounce"
                  style={{ animationDelay: '0.2s' }}
                ></div>
                <div
                  className="w-2 h-2 bg-gray-400 rounded-full animate-bounce"
                  style={{ animationDelay: '0.4s' }}
                ></div>
              </div>
            </div>
          </div>
        )}

        <div ref={messagesEndRef} />
      </div>

      {/* Input Area */}
      <form onSubmit={handleSubmit} className="border-t border-gray-700 p-4 bg-gray-800/50">
        <div className="flex items-end space-x-2 max-w-4xl mx-auto">
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Ask me anything..."
            className="flex-1 bg-gray-900 text-white border border-gray-700 rounded-lg px-4 py-3
                       resize-none focus:outline-none focus:ring-2 focus:ring-blue-500
                       focus:border-transparent min-h-[60px] max-h-[200px]"
            rows={1}
            disabled={isLoading}
          />
          <button
            type="submit"
            disabled={!input.trim() || isLoading}
            className="bg-blue-600 hover:bg-blue-700 text-white p-3 rounded-lg
                       disabled:opacity-50 disabled:cursor-not-allowed transition-colors
                       flex items-center justify-center"
          >
            <Send size={20} />
          </button>
        </div>
      </form>
    </div>
  );
}
