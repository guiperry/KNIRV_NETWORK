// Error Inference Notification System
// Provides intelligent error analysis through LLM chat modal

import React, { useState, useEffect, useRef } from 'react';
import { createPortal } from 'react-dom';
import { errorHandler } from '../utils/errorHandler';

// Bell notification icon component
export const ErrorNotificationBell = ({ className = '' }) => {
  const [errors, setErrors] = useState([]);
  const [hasNewErrors, setHasNewErrors] = useState(false);
  const [showModal, setShowModal] = useState(false);

  useEffect(() => {
    // Subscribe to new errors
    const unsubscribeErrors = errorHandler.subscribe((error) => {
      setErrors(prev => [error, ...prev.slice(0, 9)]); // Keep last 10 errors
      setHasNewErrors(true);
    });

    // Subscribe to inference results
    const unsubscribeInference = errorHandler.subscribeToInference((error) => {
      setErrors(prev => prev.map(e => e.id === error.id ? error : e));
    });

    // Refresh errors periodically to catch deletions
    const refreshErrors = () => {
      setErrors(errorHandler.getAllErrors().slice(0, 10));
    };

    // Load existing errors
    refreshErrors();

    // Set up periodic refresh to catch error deletions
    const refreshInterval = setInterval(refreshErrors, 1000);

    return () => {
      unsubscribeErrors();
      unsubscribeInference();
      clearInterval(refreshInterval);
    };
  }, []);

  const criticalErrors = errors.filter(e => e.severity === 'critical' || e.severity === 'high');
  const errorsNeedingInference = errors.filter(e => !e.inferenceRequested && (e.severity === 'high' || e.severity === 'critical'));

  const handleBellClick = () => {
    setShowModal(true);
    setHasNewErrors(false);
  };

  return (
    <>
      <div className={`error-notification-bell relative ${className}`}>
        <button
          onClick={handleBellClick}
          className={`p-2 rounded-full transition-colors ${
            criticalErrors.length > 0
              ? 'text-red-600 hover:bg-red-50'
              : errors.length > 0
                ? 'text-yellow-600 hover:bg-yellow-50'
                : 'text-gray-400 hover:bg-gray-50'
          }`}
          title={`${errors.length} system errors`}
        >
          <svg className="w-6 h-6" fill="currentColor" viewBox="0 0 20 20">
            <path d="M10 2a6 6 0 00-6 6v3.586l-.707.707A1 1 0 004 14h12a1 1 0 00.707-1.707L16 11.586V8a6 6 0 00-6-6zM10 18a3 3 0 01-3-3h6a3 3 0 01-3 3z" />
          </svg>
          
          {/* Error count badge */}
          {errors.length > 0 && (
            <span className={`absolute -top-1 -right-1 inline-flex items-center justify-center px-2 py-1 text-xs font-bold leading-none text-white transform translate-x-1/2 -translate-y-1/2 rounded-full ${
              criticalErrors.length > 0 ? 'bg-red-600' : 'bg-yellow-600'
            }`}>
              {errors.length > 99 ? '99+' : errors.length}
            </span>
          )}
          
          {/* New error indicator */}
          {hasNewErrors && (
            <span className="absolute top-0 right-0 block w-3 h-3 bg-red-500 border-2 border-white rounded-full animate-pulse"></span>
          )}
        </button>
      </div>

      {/* Error Analysis Modal */}
      {showModal && createPortal(
        <ErrorAnalysisModal
          errors={errors}
          onClose={() => setShowModal(false)}
          onRequestInference={(errorId) => errorHandler.requestInference(errorId)}
        />,
        document.body
      )}
    </>
  );
};

// Error Analysis Modal Component
const ErrorAnalysisModal = ({ errors, onClose, onRequestInference }) => {
  const [selectedError, setSelectedError] = useState(null);
  const [chatMessages, setChatMessages] = useState([]);
  const [inputMessage, setInputMessage] = useState('');
  const [isAnalyzing, setIsAnalyzing] = useState(false);
  const chatMessagesEndRef = useRef(null);

  const handleDeleteError = (errorId) => {
    // Clear the error from the error handler
    errorHandler.clearError(errorId);

    // If the deleted error was selected, clear selection
    if (selectedError?.id === errorId) {
      setSelectedError(null);
      setChatMessages([]);
    }
  };

  useEffect(() => {
    if (errors.length > 0 && !selectedError) {
      // Auto-select the most recent critical error
      const criticalError = errors.find(e => e.severity === 'critical' || e.severity === 'high');
      setSelectedError(criticalError || errors[0]);
    }
  }, [errors, selectedError]);

  useEffect(() => {
    if (selectedError?.inferenceResult) {
      // Initialize chat with inference result
      setChatMessages([
        {
          type: 'system',
          content: `I've analyzed the error "${selectedError.message}" and here's what I found:`,
          timestamp: new Date(),
        },
        {
          type: 'assistant',
          content: selectedError.inferenceResult.analysis,
          timestamp: new Date(),
          fixes: selectedError.inferenceResult.suggestedFixes,
          confidence: selectedError.inferenceResult.confidence,
        }
      ]);
    } else {
      setChatMessages([]);
    }
  }, [selectedError]);

  // Subscribe to inference updates for the selected error
  useEffect(() => {
    if (!selectedError) return;

    const unsubscribeInference = errorHandler.subscribeToInference((updatedError) => {
      // Check if this is the same error we're currently viewing
      if (updatedError.id === selectedError.id && updatedError.inferenceResult) {
        // Update chat messages with the new inference result
        setChatMessages([
          {
            type: 'system',
            content: `I've analyzed the error "${updatedError.message}" and here's what I found:`,
            timestamp: new Date(),
          },
          {
            type: 'assistant',
            content: updatedError.inferenceResult.analysis,
            timestamp: new Date(),
            fixes: updatedError.inferenceResult.suggestedFixes,
            confidence: updatedError.inferenceResult.confidence,
          }
        ]);
      }
    });

    return unsubscribeInference;
  }, [selectedError?.id]);

  // Auto-scroll to bottom when new messages are added
  useEffect(() => {
    if (chatMessagesEndRef.current) {
      chatMessagesEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [chatMessages]);

  const handleAnalyzeError = async () => {
    if (!selectedError || selectedError.inferenceRequested) return;

    setIsAnalyzing(true);
    setChatMessages(prev => [...prev, {
      type: 'system',
      content: 'Analyzing error... Please wait.',
      timestamp: new Date(),
    }]);

    try {
      await onRequestInference(selectedError.id);
    } catch (error) {
      setChatMessages(prev => [...prev, {
        type: 'system',
        content: 'Failed to analyze error. Please try again.',
        timestamp: new Date(),
      }]);
    } finally {
      setIsAnalyzing(false);
    }
  };

  const handleSendMessage = async () => {
    if (!inputMessage.trim() || !selectedError) return;

    const userMessage = {
      type: 'user',
      content: inputMessage,
      timestamp: new Date(),
    };

    setChatMessages(prev => [...prev, userMessage]);
    setInputMessage('');
    
    // Add loading message
    setChatMessages(prev => [...prev, {
      type: 'system',
      content: 'Processing your question...',
      timestamp: new Date(),
    }]);

    try {
      // Get the correct API base URL for Electron vs web
      const apiBaseUrl = typeof window !== 'undefined' && window.electronAPI?.isElectron
        ? 'http://localhost:8081'
        : '';

      // Call the error chat API with the inference engine
      const response = await fetch(`${apiBaseUrl}/api/v1/inference/chat-error`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          error_id: selectedError.id,
          message: inputMessage,
          error_details: {
            type: selectedError.type,
            severity: selectedError.severity,
            message: selectedError.message,
            details: selectedError.details,
            code: selectedError.code,
            technicalMessage: selectedError.technicalMessage,
          },
          conversation_history: chatMessages.map(msg => ({
            type: msg.type,
            content: msg.content,
            timestamp: msg.timestamp.toISOString(),
          })),
        }),
      });

      if (!response.ok) {
        throw new Error(`Chat API failed: ${response.statusText}`);
      }

      const result = await response.json();
      
      // Remove the loading message
      setChatMessages(prev => prev.filter(msg => 
        !(msg.type === 'system' && msg.content === 'Processing your question...')
      ));

      // Add the AI response
      setChatMessages(prev => [...prev, {
        type: 'assistant',
        content: result.response,
        timestamp: new Date(),
      }]);
    } catch (error) {
      console.error('Error chat API error:', error);
      
      // Remove the loading message
      setChatMessages(prev => prev.filter(msg => 
        !(msg.type === 'system' && msg.content === 'Processing your question...')
      ));
      
      // Add error message
      setChatMessages(prev => [...prev, {
        type: 'system',
        content: 'Sorry, I encountered an error while processing your question. Please try again.',
        timestamp: new Date(),
      }]);
    }
  };

  const getSeverityColor = (severity) => {
    switch (severity) {
      case 'critical': return 'text-red-600 bg-red-50';
      case 'high': return 'text-red-500 bg-red-50';
      case 'medium': return 'text-yellow-600 bg-yellow-50';
      case 'low': return 'text-blue-600 bg-blue-50';
      default: return 'text-gray-600 bg-gray-50';
    }
  };

  return (
    <div
      className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center"
      style={{
        position: 'fixed',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        zIndex: 99999,
        margin: 0,
        padding: 0
      }}
    >
      <div className="bg-slate-800 rounded-lg shadow-xl w-full max-w-4xl h-3/4 flex flex-col mx-4" style={{ maxHeight: '90vh' }}>
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-slate-700">
          <h2 className="text-lg font-semibold text-white">
            🤖 AI Error Analysis Assistant
          </h2>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-white transition-colors"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div className="flex flex-1 overflow-hidden">
          {/* Error List Sidebar */}
          <div className="w-1/3 border-r border-slate-700 bg-slate-900/50 overflow-y-auto">
            <div className="p-4">
              <h3 className="text-sm font-medium text-slate-300 mb-3">Recent Errors</h3>
              <div className="space-y-2">
                {errors.map((error) => (
                  <div
                    key={error.id}
                    className={`relative p-3 rounded-lg border transition-colors ${
                      selectedError?.id === error.id
                        ? 'border-blue-500 bg-blue-500/20'
                        : 'border-slate-600 bg-slate-800/50 hover:bg-slate-700/50'
                    }`}
                  >
                    <button
                      onClick={() => setSelectedError(error)}
                      className="w-full text-left"
                    >
                      <div className="flex items-center justify-between mb-1">
                        <span className={`text-xs px-2 py-1 rounded-full ${getSeverityColor(error.severity)}`}>
                          {error.severity}
                        </span>
                        <span className="text-xs text-slate-400">
                          {error.timestamp.toLocaleTimeString()}
                        </span>
                      </div>
                      <div className="text-sm font-medium text-white truncate">
                        {error.message}
                      </div>
                      <div className="text-xs text-slate-400 truncate">
                        {error.type} • {error.code || 'No code'}
                      </div>
                      {error.inferenceResult && (
                        <div className="text-xs text-green-400 mt-1">
                          ✓ Analysis available
                        </div>
                      )}
                    </button>
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        handleDeleteError(error.id);
                      }}
                      className="absolute top-2 right-2 p-1 text-slate-400 hover:text-red-400 transition-colors"
                      title="Delete error"
                    >
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                      </svg>
                    </button>
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* Chat Interface */}
          <div className="flex-1 flex flex-col">
            {selectedError ? (
              <>
                {/* Error Details Header */}
                <div className="p-4 border-b border-slate-700 bg-slate-900/30">
                  <div className="flex items-center justify-between">
                    <div>
                      <h4 className="font-medium text-white">{selectedError.message}</h4>
                      <p className="text-sm text-slate-400">{selectedError.userMessage}</p>
                    </div>
                    {!selectedError.inferenceResult && !selectedError.inferenceRequested && (
                      <button
                        onClick={handleAnalyzeError}
                        disabled={isAnalyzing}
                        className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 transition-colors"
                      >
                        {isAnalyzing ? 'Analyzing...' : 'Analyze Error'}
                      </button>
                    )}
                  </div>
                </div>

                {/* Chat Messages */}
                <div className="flex-1 overflow-y-auto p-4 space-y-4">
                  {chatMessages.map((message, index) => (
                    <div
                      key={index}
                      className={`flex ${message.type === 'user' ? 'justify-end' : 'justify-start'}`}
                    >
                      <div
                        className={`max-w-3/4 p-3 rounded-lg ${
                          message.type === 'user'
                            ? 'bg-blue-600 text-white'
                            : message.type === 'system'
                            ? 'bg-slate-700 text-slate-300'
                            : 'bg-slate-800 text-slate-200 border border-slate-600'
                        }`}
                      >
                        <div className="text-sm">{message.content}</div>
                        {message.fixes && (
                          <div className="mt-3">
                            <div className="text-xs font-medium mb-2">Suggested Fixes:</div>
                            <ol className="text-xs space-y-1">
                              {message.fixes.map((fix, i) => (
                                <li key={i} className="flex items-start">
                                  <span className="mr-2">{i + 1}.</span>
                                  <span>{fix}</span>
                                </li>
                              ))}
                            </ol>
                            {message.confidence && (
                              <div className="text-xs text-gray-600 mt-2">
                                Confidence: {Math.round(message.confidence * 100)}%
                              </div>
                            )}
                          </div>
                        )}
                        <div className="text-xs opacity-70 mt-1">
                          {message.timestamp.toLocaleTimeString()}
                        </div>
                      </div>
                    </div>
                  ))}
                  {/* Invisible element to scroll to */}
                  <div ref={chatMessagesEndRef} />
                </div>

                {/* Chat Input */}
                <div className="p-4 border-t border-slate-700">
                  <div className="flex space-x-2">
                    <input
                      type="text"
                      value={inputMessage}
                      onChange={(e) => setInputMessage(e.target.value)}
                      onKeyPress={(e) => e.key === 'Enter' && handleSendMessage()}
                      placeholder="Ask about this error or request additional help..."
                      className="flex-1 px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    />
                    <button
                      onClick={handleSendMessage}
                      disabled={!inputMessage.trim()}
                      className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 transition-colors"
                    >
                      Send
                    </button>
                  </div>
                </div>
              </>
            ) : (
              <div className="flex-1 flex items-center justify-center text-slate-400">
                Select an error to start analysis
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
