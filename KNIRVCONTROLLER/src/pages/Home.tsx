import React, { useState, useEffect, useRef } from 'react';
import { Bot, Shield, Activity, TrendingUp, QrCode, Mic, Eye, Calendar, Send, User, BookmarkPlus, Image as ImageIcon } from 'lucide-react';
import ManagerLayout from '../components/ManagerLayout';
import StatsCard from '../components/StatsCard';
import AgentCard from '../components/AgentCard';
import QRScanner from '../components/QRScanner';
import VoiceProcessor from '../components/VoiceProcessor';
import AnalyticsDashboard from '../components/AnalyticsDashboard';
import TaskScheduler from '../components/TaskScheduler';
import UDCManager from '../components/UDCManager';
import PerformanceMonitor from '../components/PerformanceMonitor';
import KNIRVANAGameVisualization from '../components/KNIRVANAGameVisualization';
import { desktopConnection } from '../services/DesktopConnection';
import { agentManagementService } from '../services/AgentManagementService';
import { ChatBrainProvider, useChatBrain } from '../contexts/ChatBrainContext';
import ReactMarkdown from 'react-markdown';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';

interface HRMProcessingResponse {
  reasoning_result: string;
  confidence: number;
  processing_time: number;
  l_module_activations: number[];
  h_module_activations: number[];
}

function HomeContent() {
  const { messages, sendMessage, isLoading, saveNote } = useChatBrain();
  const [input, setInput] = useState('');
  const [showInputOptions, setShowInputOptions] = useState(false);
  const [voiceActive, setVoiceActive] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);

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
    } catch (error) {
      console.error('Failed to save note:', error);
    }
  };

  const handleVoiceCommand = async (command: string, confidence: number) => {
    console.log('Voice command:', command, 'Confidence:', confidence);
    await sendMessage(`Voice input: ${command} (confidence: ${confidence})`);
    setVoiceActive(false);
  };

  const handleAudioData = (_audioData: Float32Array) => {
    console.log('Audio data received');
  };

  const handleImageUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      const file = e.target.files[0];
      sendMessage(`Image input: ${file.name}`);
    }
  };

  return (
    <div className="flex-1 flex flex-col bg-gray-900">
      {/* Messages Area */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {messages.length === 0 && !showInputOptions ? (
          <div className="flex items-center justify-center h-full">
            <div className="text-center text-gray-400 max-w-md">
              <h2 className="text-2xl font-bold mb-2 text-white">Welcome to KNIRV</h2>
              <p className="text-gray-500 mb-8">
                Click the central icon below to start interacting with your AI assistant
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
                      code({ inline, className, children, ...props }: React.ComponentPropsWithoutRef<'code'> & { inline?: boolean }) {
                        const match = /language-(\w+)/.exec(className || '');
                        return !inline && match ? (
                          <SyntaxHighlighter
                            style={vscDarkPlus}
                            language={match[1]}
                            PreTag="div"
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

      {/* Input Options */}
      {showInputOptions && (
        <div className="border-t border-gray-700 p-4 bg-gray-800/50">
          <div className="flex items-center justify-around mb-4">
            <button
              onClick={() => setVoiceActive(!voiceActive)}
              className={`p-3 rounded-lg transition-colors ${
                voiceActive ? 'bg-green-600 text-white' : 'bg-gray-700 hover:bg-gray-600 text-gray-300'
              }`}
            >
              <Mic size={24} />
            </button>
            <button
              onClick={() => setShowInputOptions(false)}
              className="p-3 rounded-lg bg-gray-700 hover:bg-gray-600 text-gray-300 transition-colors"
            >
              <Send size={24} />
            </button>
            <label className="cursor-pointer p-3 rounded-lg bg-gray-700 hover:bg-gray-600 text-gray-300 transition-colors">
              <ImageIcon size={24} />
              <input
                type="file"
                accept="image/*"
                onChange={handleImageUpload}
                className="hidden"
              />
            </label>
          </div>

          {/* Text Input */}
          <form onSubmit={handleSubmit} className="flex items-end space-x-2 max-w-4xl mx-auto">
            <textarea
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Type your message..."
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
          </form>
        </div>
      )}

      {/* Central Input Receiver */}
      {!showInputOptions && messages.length === 0 && (
        <div className="absolute bottom-24 left-1/2 transform -translate-x-1/2">
          <button
            onClick={() => setShowInputOptions(true)}
            className="w-20 h-20 bg-gradient-to-br from-purple-600 to-cyan-600 rounded-full
                       flex items-center justify-center shadow-xl hover:scale-110
                       transition-transform duration-300"
          >
            <Bot size={40} className="text-white" />
          </button>
        </div>
      )}

      {/* Voice Processor */}
      {voiceActive && (
        <VoiceProcessor
          onVoiceCommand={handleVoiceCommand}
          onAudioData={handleAudioData}
          isActive={voiceActive}
        />
      )}


    </div>
  );
}

interface ActionButtonProps {
  icon: React.ComponentType<{ className?: string }>;
  title: string;
  description: string;
  onClick?: () => void;
}

function ActionButton({ icon: Icon, title, description, onClick }: ActionButtonProps) {
  return (
    <button className="relative group" onClick={onClick}>
      <div className="absolute -inset-0.5 bg-gradient-to-r from-purple-600/30 to-cyan-600/30 rounded-xl blur opacity-20 group-hover:opacity-50 group-active:opacity-75 transition duration-200"></div>
      <div className="relative bg-slate-800/60 backdrop-blur-xl rounded-xl p-4 border border-slate-700/50 hover:border-purple-500/50 group-active:scale-95 transition-all text-left">
        <div className="flex items-center space-x-3">
          <div className="w-10 h-10 bg-gradient-to-br from-purple-500/20 to-cyan-500/20 rounded-lg flex items-center justify-center border border-purple-500/20">
            <Icon className="w-5 h-5 text-purple-400" />
          </div>
          <div>
            <h4 className="font-medium text-white text-sm">{title}</h4>
            <p className="text-xs text-slate-400">{description}</p>
          </div>
        </div>
      </div>
    </button>
  );
}

export default function Home() {
  const [showQRScanner, setShowQRScanner] = useState(false);
  const [connectionStatus, setConnectionStatus] = useState(desktopConnection.getConnectionStatus());
  const [hrmResponse, setHrmResponse] = useState<HRMProcessingResponse | null>(null);
  const [showAnalytics, setShowAnalytics] = useState(false);
  const [showTaskScheduler, setShowTaskScheduler] = useState(false);
  const [showUDCManager, setShowUDCManager] = useState(false);
  const [showPerformanceMonitor, setShowPerformanceMonitor] = useState(false);
  const [showKNIRVANAGame, setShowKNIRVANAGame] = useState(false);

  useEffect(() => {
    desktopConnection.setConnectionChangeHandler(setConnectionStatus);
    desktopConnection.setHRMResponseHandler(setHrmResponse);
  }, []);

  const handleQRScan = async (qrData: string) => {
    try {
      const parsedData = JSON.parse(qrData);
      const success = await desktopConnection.connectToDesktop(parsedData);

      if (success) {
        setShowQRScanner(false);
        console.log('Successfully connected to desktop');
      }
    } catch (error) {
      console.error('Failed to connect:', error);
    }
  };

  const handleDeployAgent = async () => {
    try {
      const agents = await agentManagementService.getAgents();
      const availableAgent = agents.find(agent => agent.status === 'Available');

      if (!availableAgent) {
        alert('No available agents found');
        return;
      }

      const deploymentId = await agentManagementService.deployAgent({
        agentId: availableAgent.agentId,
        targetNRV: undefined,
        configuration: {},
        resources: {
          memory: availableAgent.metadata.requirements.memory,
          cpu: availableAgent.metadata.requirements.cpu,
          timeout: 300000
        }
      });

      console.log('Agent deployed successfully:', deploymentId);
      alert(`Agent ${availableAgent.name} deployed successfully!`);
    } catch (error) {
      console.error('Failed to deploy agent:', error);
      alert(`Failed to deploy agent: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  };

  const handleViewAnalytics = () => {
    setShowAnalytics(true);
  };

  const handleScheduleTask = () => {
    setShowTaskScheduler(true);
  };

  const handleRenewUDC = () => {
    setShowUDCManager(true);
  };

  const handleKNIRVANAGame = () => {
    setShowKNIRVANAGame(true);
  };

  const agents = [
    {
      name: 'CodeT5-Alpha',
      status: 'active' as const,
      tasks: 12,
      performance: 94,
      lastActive: '2 min ago'
    },
    {
      name: 'SEAL-Beta',
      status: 'active' as const,
      tasks: 8,
      performance: 87,
      lastActive: '5 min ago'
    },
    {
      name: 'LoRA-Gamma',
      status: 'idle' as const,
      tasks: 0,
      performance: 91,
      lastActive: '1 hour ago'
    },
    {
      name: 'NRN-Delta',
      status: 'error' as const,
      tasks: 0,
      performance: 78,
      lastActive: '3 hours ago'
    }
  ];

  return (
    <ChatBrainProvider>
      <ManagerLayout>
        <div className="p-4 pb-24 space-y-6">
          {/* Welcome Section */}
          <div className="text-center py-6">
            <h2 className="text-2xl font-bold bg-gradient-to-r from-purple-400 to-cyan-400 bg-clip-text text-transparent mb-2">
              KNIRV Mobile Tool
            </h2>
            <p className="text-slate-400 text-sm">
              {connectionStatus.connected
                ? `Connected to Desktop ${connectionStatus.desktop_id?.slice(0, 8)}...`
                : 'Enhanced mobile client with HRM integration'
              }
            </p>
          </div>

          {/* Desktop Connection Status */}
          <div className={`relative group ${connectionStatus.connected ? 'mb-4' : 'mb-6'}`}>
            <div className={`absolute -inset-0.5 bg-gradient-to-r ${
              connectionStatus.connected
                ? 'from-green-600/50 to-cyan-600/50'
                : 'from-orange-600/50 to-red-600/50'
            } rounded-xl blur opacity-30 group-hover:opacity-60 transition duration-300`}></div>
            <div className={`relative bg-slate-800/80 backdrop-blur-xl rounded-xl p-4 border ${
              connectionStatus.connected
                ? 'border-green-500/30'
                : 'border-orange-500/30'
            }`}>
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-3">
                  <div className={`w-3 h-3 rounded-full ${
                    connectionStatus.connected
                      ? 'bg-green-400 animate-pulse'
                      : 'bg-orange-400'
                  }`}></div>
                  <div>
                    <h3 className="font-semibold text-white">
                      {connectionStatus.connected ? 'Desktop Connected' : 'Desktop Disconnected'}
                    </h3>
                    <p className={`text-sm ${
                      connectionStatus.connected ? 'text-green-400' : 'text-orange-400'
                    }`}>
                      {connectionStatus.connected
                        ? 'HRM cognitive processing available'
                        : 'Scan QR code to connect'
                      }
                    </p>
                  </div>
                </div>
                <button
                  onClick={() => setShowQRScanner(true)}
                  className="p-2 bg-purple-500/20 hover:bg-purple-500/30 rounded-lg transition-colors"
                >
                  <QrCode size={20} className="text-purple-400" />
                </button>
              </div>
            </div>
          </div>

          {/* Chat Interface */}
          <HomeContent />

          {/* Stats Grid */}
          <div className="grid grid-cols-2 gap-4">
            <StatsCard
              title="Active Agents"
              value={2}
              change="+25% from last hour"
              icon={Bot}
              trend="up"
            />
            <StatsCard
              title="NRN Balance"
              value="1,247"
              change="-12 NRN consumed"
              icon={TrendingUp}
              trend="down"
            />
            <StatsCard
              title="UDC Status"
              value="Valid"
              change="Expires in 7 days"
              icon={Shield}
              trend="neutral"
            />
            <StatsCard
              title="System Health"
              value="98%"
              change="+2% improvement"
              icon={Activity}
              trend="up"
            />
          </div>

          {/* SEAL Loop Status */}
          <div className="relative group">
            <div className="absolute -inset-0.5 bg-gradient-to-r from-green-600/50 to-cyan-600/50 rounded-xl blur opacity-30 group-hover:opacity-60 transition duration-300"></div>
            <div className="relative bg-slate-800/80 backdrop-blur-xl rounded-xl p-4 border border-green-500/30">
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-3">
                  <div className="w-3 h-3 bg-green-400 rounded-full animate-pulse"></div>
                  <div>
                    <h3 className="font-semibold text-white">SEAL Loop Active</h3>
                    <p className="text-sm text-green-400">Continuous optimization running</p>
                  </div>
                </div>
                <div className="text-right">
                  <p className="text-sm text-slate-400">Next cycle</p>
                  <p className="text-xs text-slate-500">in 3m 24s</p>
                </div>
              </div>
            </div>
          </div>

          {/* Agents Section */}
          <div>
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-white">Your Agents</h3>
              <button className="text-sm text-purple-400 hover:text-purple-300 transition-colors">
                View All
              </button>
            </div>
            
            <div className="space-y-4">
              {agents.map((agent) => (
                <AgentCard key={agent.name} {...agent} />
              ))}
            </div>
          </div>

          {/* Quick Actions */}
          <div>
            <h3 className="text-lg font-semibold text-white mb-4">Quick Actions</h3>
            <div className="grid grid-cols-2 gap-3">
              <ActionButton
                icon={Bot}
                title="Deploy Agent"
                description="Launch new AI agent"
                onClick={handleDeployAgent}
              />
              <ActionButton
                icon={TrendingUp}
                title="View Analytics"
                description="Performance insights"
                onClick={handleViewAnalytics}
              />
              <ActionButton
                icon={Calendar}
                title="Schedule Task"
                description="Automate workflows"
                onClick={handleScheduleTask}
              />
              <ActionButton
                icon={Shield}
                title="Renew UDC"
                description="Extend certificate"
                onClick={handleRenewUDC}
              />
              <ActionButton
                icon={Activity}
                title="Performance Monitor"
                description="System optimization"
                onClick={() => setShowPerformanceMonitor(true)}
              />
              <ActionButton
                icon={Eye}
                title="KNIRVANA Graph"
                description="Game mechanics visualization"
                onClick={handleKNIRVANAGame}
              />
            </div>
          </div>

          {/* HRM Response Display */}
          {hrmResponse && (
            <div className="relative group">
              <div className="absolute -inset-0.5 bg-gradient-to-r from-cyan-600/50 to-purple-600/50 rounded-xl blur opacity-30 group-hover:opacity-60 transition duration-300"></div>
              <div className="relative bg-slate-800/80 backdrop-blur-xl rounded-xl p-4 border border-cyan-500/30">
                <h3 className="font-semibold text-white mb-2">HRM Cognitive Response</h3>
                <p className="text-sm text-cyan-400 mb-2">{hrmResponse.reasoning_result}</p>
                <div className="grid grid-cols-2 gap-4 text-xs">
                  <div>
                    <span className="text-slate-400">Confidence:</span>
                    <span className="ml-2 text-white">{(hrmResponse.confidence * 100).toFixed(1)}%</span>
                  </div>
                  <div>
                    <span className="text-slate-400">Processing:</span>
                    <span className="ml-2 text-white">{hrmResponse.processing_time.toFixed(1)}ms</span>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* QR Scanner Modal */}
        <QRScanner
          isOpen={showQRScanner}
          onScan={handleQRScan}
          onClose={() => setShowQRScanner(false)}
        />

        {/* Analytics Dashboard Modal */}
        <AnalyticsDashboard
          isOpen={showAnalytics}
          onClose={() => setShowAnalytics(false)}
        />

        {/* Task Scheduler Modal */}
        <TaskScheduler
          isOpen={showTaskScheduler}
          onClose={() => setShowTaskScheduler(false)}
        />

        {/* UDC Manager Modal */}
        <UDCManager
          isOpen={showUDCManager}
          onClose={() => setShowUDCManager(false)}
        />

        {/* Performance Monitor Modal */}
        <PerformanceMonitor
          isOpen={showPerformanceMonitor}
          onClose={() => setShowPerformanceMonitor(false)}
        />

        {/* KNIRVANA Game Visualization Modal */}
        <div className={`fixed inset-0 z-50 ${showKNIRVANAGame ? 'block' : 'hidden'}`}>
          <div className="absolute inset-0 bg-black/50 backdrop-blur-sm" onClick={() => setShowKNIRVANAGame(false)} />
          <div className="absolute inset-4 max-w-6xl mx-auto">
            <div className="bg-slate-900/95 backdrop-blur-xl rounded-xl overflow-hidden shadow-2xl">
              <div className="flex justify-between items-center p-4 border-b border-slate-700">
                <h2 className="text-xl font-semibold text-white">KNIRVANA Game Bridge</h2>
                <button
                  onClick={() => setShowKNIRVANAGame(false)}
                  className="text-slate-400 hover:text-white transition-colors"
                >
                  <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
              <div className="max-h-[80vh] overflow-y-auto p-4">
                <KNIRVANAGameVisualization />
              </div>
            </div>
          </div>
        </div>
      </ManagerLayout>
    </ChatBrainProvider>
  );
}
