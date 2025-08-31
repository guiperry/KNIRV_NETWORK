import * as React from 'react';
import { useState, useEffect } from 'react';
import { Bot, Shield, Activity, TrendingUp, QrCode, Mic, Eye, Calendar } from 'lucide-react';
import ManagerLayout from '../components/ManagerLayout';
import StatsCard from '../components/StatsCard';
import AgentCard from '../components/AgentCard';
import QRScanner from '../components/QRScanner';
import VoiceProcessor from '../components/VoiceProcessor';
import VisualProcessor from '../components/VisualProcessor';
import AnalyticsDashboard from '../components/AnalyticsDashboard';
import TaskScheduler from '../components/TaskScheduler';
import UDCManager from '../components/UDCManager';
import PerformanceMonitor from '../components/PerformanceMonitor';
import { desktopConnection } from '../services/DesktopConnection';

interface HRMProcessingResponse {
  reasoning_result: string;
  confidence: number;
  processing_time: number;
  l_module_activations: number[];
  h_module_activations: number[];
}
// import { agentManagementService } from '../services/AgentManagementService';

export default function Home() {
  const [showQRScanner, setShowQRScanner] = useState(false);
  const [voiceActive, setVoiceActive] = useState(false);
  const [visualActive, setVisualActive] = useState(false);
  const [connectionStatus, setConnectionStatus] = useState(desktopConnection.getConnectionStatus());
  const [hrmResponse, setHrmResponse] = useState<HRMProcessingResponse | null>(null);
  const [showAnalytics, setShowAnalytics] = useState(false);
  const [showTaskScheduler, setShowTaskScheduler] = useState(false);
  const [showUDCManager, setShowUDCManager] = useState(false);
  const [showPerformanceMonitor, setShowPerformanceMonitor] = useState(false);

  useEffect(() => {
    // Set up desktop connection event handlers
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

  const handleVoiceCommand = async (command: string, confidence: number) => {
    console.log('Voice command:', command, 'Confidence:', confidence);

    // Send voice data to HRM for processing
    if (connectionStatus.connected) {
      await desktopConnection.sendHRMRequest({
        sensory_data: [confidence],
        context: 'voice_command',
        task_type: 'voice_processing'
      });
    }
  };

  const handleAudioData = (audioData: Float32Array) => {
    // Process audio data for HRM
    const audioArray = Array.from(audioData.slice(0, 10)); // Take first 10 values

    if (connectionStatus.connected) {
      desktopConnection.sendHRMRequest({
        sensory_data: audioArray,
        context: 'audio_analysis',
        task_type: 'audio_processing'
      });
    }
  };

  const handleVisualData = (imageData: ImageData) => {
    // Process visual data for HRM (sample pixels)
    const pixels = [];
    for (let i = 0; i < imageData.data.length; i += 4000) { // Sample every 4000th pixel
      pixels.push(imageData.data[i] / 255); // Normalize to 0-1
      if (pixels.length >= 20) break; // Limit to 20 samples
    }

    if (connectionStatus.connected && pixels.length > 0) {
      desktopConnection.sendHRMRequest({
        sensory_data: pixels,
        context: 'visual_analysis',
        task_type: 'visual_processing'
      });
    }
  };

  const handleObjectDetection = (objects: unknown[]) => {
    console.log('Detected objects:', objects);
  };

  // Action button handlers
  const handleDeployAgent = () => {
    // Navigate to agent management or open agent deployment modal
    // For now, we'll show a simple alert - in a full implementation,
    // this would open an agent deployment interface
    alert('Deploy Agent functionality - would open agent deployment interface');
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

        {/* Mobile Tool Controls */}
        <div className="grid grid-cols-3 gap-3 mb-6">
          <button
            onClick={() => setShowQRScanner(true)}
            className="relative group"
          >
            <div className="absolute -inset-0.5 bg-gradient-to-r from-purple-600/30 to-cyan-600/30 rounded-xl blur opacity-20 group-hover:opacity-50 transition duration-200"></div>
            <div className="relative bg-slate-800/60 backdrop-blur-xl rounded-xl p-4 border border-slate-700/50 hover:border-purple-500/50 transition-all text-center">
              <QrCode className="w-6 h-6 text-purple-400 mx-auto mb-2" />
              <p className="text-xs text-slate-400">QR Scan</p>
            </div>
          </button>

          <button
            onClick={() => setVoiceActive(!voiceActive)}
            className="relative group"
          >
            <div className="absolute -inset-0.5 bg-gradient-to-r from-green-600/30 to-blue-600/30 rounded-xl blur opacity-20 group-hover:opacity-50 transition duration-200"></div>
            <div className={`relative bg-slate-800/60 backdrop-blur-xl rounded-xl p-4 border transition-all text-center ${
              voiceActive ? 'border-green-500/50' : 'border-slate-700/50 hover:border-green-500/50'
            }`}>
              <Mic className={`w-6 h-6 mx-auto mb-2 ${voiceActive ? 'text-green-400' : 'text-green-400'}`} />
              <p className="text-xs text-slate-400">Voice</p>
            </div>
          </button>

          <button
            onClick={() => setVisualActive(!visualActive)}
            className="relative group"
          >
            <div className="absolute -inset-0.5 bg-gradient-to-r from-blue-600/30 to-purple-600/30 rounded-xl blur opacity-20 group-hover:opacity-50 transition duration-200"></div>
            <div className={`relative bg-slate-800/60 backdrop-blur-xl rounded-xl p-4 border transition-all text-center ${
              visualActive ? 'border-blue-500/50' : 'border-slate-700/50 hover:border-blue-500/50'
            }`}>
              <Eye className={`w-6 h-6 mx-auto mb-2 ${visualActive ? 'text-blue-400' : 'text-blue-400'}`} />
              <p className="text-xs text-slate-400">Visual</p>
            </div>
          </button>
        </div>

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

        {/* Voice Processor */}
        {voiceActive && (
          <VoiceProcessor
            onVoiceCommand={handleVoiceCommand}
            onAudioData={handleAudioData}
            isActive={voiceActive}
          />
        )}

        {/* Visual Processor */}
        {visualActive && (
          <VisualProcessor
            onVisualData={handleVisualData}
            onObjectDetection={handleObjectDetection}
            isActive={visualActive}
          />
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
    </ManagerLayout>
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
