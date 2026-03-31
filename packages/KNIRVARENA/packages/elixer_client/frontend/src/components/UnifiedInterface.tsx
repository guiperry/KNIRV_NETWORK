import React, { useState, useEffect } from 'react';
import { ComponentBridge, ComponentMessage, SystemState } from '@shared/ComponentBridge';
import { QrCode, Terminal, Wallet, Settings, Menu, X } from 'lucide-react';

interface UnifiedInterfaceProps {
  bridge: ComponentBridge;
}

interface ReceiverState {
  loaded: boolean;
  error?: string;
  cognitiveActive: boolean;
  currentSkill?: string;
}

const UnifiedInterface: React.FC<UnifiedInterfaceProps> = ({ bridge }) => {
  const [systemState, setSystemState] = useState<SystemState>(bridge.getState());
  const [receiverState, setReceiverState] = useState<ReceiverState>({
    loaded: false,
    cognitiveActive: false
  });
  const [showMobileMenu, setShowMobileMenu] = useState(false);
  const [showCLI, setShowCLI] = useState(false);
  const [showQRScanner, setShowQRScanner] = useState(false);

  useEffect(() => {
    // Subscribe to system state updates
    bridge.onMessage('*', (_message: ComponentMessage) => {
      setSystemState(bridge.getState());
    });

    // Subscribe to receiver-specific updates
    bridge.onMessage('receiver_status', (message: ComponentMessage) => {
      setReceiverState(prev => ({
        ...prev,
        ...(typeof message.payload === 'object' && message.payload !== null ? message.payload as Partial<ReceiverState> : {})
      }));
    });

    // Request initial receiver status
    bridge.sendMessage('status_request', 'receiver', 'get_status');

    const waitForComponent = (componentName: string, timeout: number): Promise<boolean> => {
      return new Promise((resolve) => {
        const startTime = Date.now();
        
        const check = () => {
          const state = bridge.getState();
          if (state.components[componentName] === 'running') {
            resolve(true);
            return;
          }
          
          if (Date.now() - startTime > timeout) {
            resolve(false);
            return;
          }
          
          setTimeout(check, 500);
        };
        
        check();
      });
    };

    // Wait for receiver component to be ready
    const checkReceiver = async () => {
      try {
        const receiverReady = await waitForComponent('receiver', 10000);
        if (receiverReady) {
          setReceiverState(prev => ({ ...prev, loaded: true }));
        } else {
          setReceiverState(prev => ({
            ...prev,
            loaded: false,
            error: 'Receiver component failed to load'
          }));
        }
      } catch (error) {
        setReceiverState(prev => ({
          ...prev,
          loaded: false,
          error: error instanceof Error ? error.message : 'Unknown error'
        }));
      }
    };

    checkReceiver();
  }, [bridge]);


  const handleQRScan = () => {
    setShowQRScanner(true);
    bridge.requestQRScan();
  };


  const handleCLIToggle = () => {
    setShowCLI(!showCLI);
    if (!showCLI) {
      bridge.sendMessage('cli_request', 'cli', 'show_terminal');
    }
  };


  const renderReceiverInterface = () => {
    if (!receiverState.loaded) {
      return (
        <div className="flex-1 flex items-center justify-center bg-gray-900">
          <div className="text-center text-white">
            {receiverState.error ? (
              <>
                <div className="text-red-500 text-xl mb-4">⚠️ Error Loading Cognitive Shell</div>
                <p className="text-gray-400">{receiverState.error}</p>
                <button 
                  onClick={() => window.location.reload()}
                  className="mt-4 px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
                >
                  Retry
                </button>
              </>
            ) : (
              <>
                <div className="animate-spin rounded-full h-16 w-16 border-b-2 border-blue-500 mx-auto mb-4"></div>
                <p>Loading Cognitive Shell...</p>
              </>
            )}
          </div>
        </div>
      );
    }

    // Embed receiver interface via iframe for now
    // In a full implementation, this would be the actual receiver components
    return (
      <div className="flex-1 relative">
        <iframe
          src="http://localhost:3002"
          className="w-full h-full border-none"
          title="KNIRV Cognitive Receiver"
          onLoad={() => setReceiverState(prev => ({ ...prev, loaded: true }))}
          onError={() => setReceiverState(prev => ({ 
            ...prev, 
            error: 'Failed to load receiver interface' 
          }))}
        />
        
        {/* Overlay controls for mobile */}
        <div className="absolute top-4 right-4 flex space-x-2 md:hidden">
          <button
            onClick={() => setShowMobileMenu(!showMobileMenu)}
            className="p-2 bg-black bg-opacity-50 text-white rounded-full"
          >
            {showMobileMenu ? <X size={20} /> : <Menu size={20} />}
          </button>
        </div>

        {/* Mobile menu overlay */}
        {showMobileMenu && (
          <div className="absolute top-0 left-0 w-full h-full bg-black bg-opacity-90 flex items-center justify-center md:hidden">
            <div className="grid grid-cols-2 gap-4 p-8">
              <button
                onClick={handleQRScan}
                className="flex flex-col items-center p-6 bg-blue-600 text-white rounded-lg"
              >
                <QrCode size={32} className="mb-2" />
                <span>QR Scan</span>
              </button>
              
              <button
                onClick={handleCLIToggle}
                className="flex flex-col items-center p-6 bg-green-600 text-white rounded-lg"
              >
                <Terminal size={32} className="mb-2" />
                <span>Terminal</span>
              </button>
              
              <button
                onClick={() => bridge.sendMessage('wallet_request', 'manager', 'show')}
                className="flex flex-col items-center p-6 bg-purple-600 text-white rounded-lg"
              >
                <Wallet size={32} className="mb-2" />
                <span>Wallet</span>
              </button>
              
              <button
                onClick={() => setShowMobileMenu(false)}
                className="flex flex-col items-center p-6 bg-gray-600 text-white rounded-lg"
              >
                <Settings size={32} className="mb-2" />
                <span>Settings</span>
              </button>
            </div>
          </div>
        )}
      </div>
    );
  };

  const renderDesktopSidebar = () => (
    <div className="hidden md:flex md:w-64 bg-gray-800 text-white flex-col">
      <div className="p-4 border-b border-gray-700">
        <h2 className="text-xl font-bold">KNIRV Controller</h2>
        <p className="text-sm text-gray-400">Unified Interface</p>
      </div>
      
      <div className="flex-1 p-4 space-y-4">
        <div className="space-y-2">
          <h3 className="text-sm font-semibold text-gray-400 uppercase">Quick Actions</h3>
          
          <button
            onClick={handleQRScan}
            className="w-full flex items-center space-x-3 p-3 bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors"
          >
            <QrCode size={20} />
            <span>QR Scanner</span>
          </button>
          
          <button
            onClick={handleCLIToggle}
            className={`w-full flex items-center space-x-3 p-3 rounded-lg transition-colors ${
              showCLI ? 'bg-green-700' : 'bg-green-600 hover:bg-green-700'
            }`}
          >
            <Terminal size={20} />
            <span>Terminal</span>
          </button>
          
          <button
            onClick={() => bridge.sendMessage('wallet_request', 'manager', 'show')}
            className="w-full flex items-center space-x-3 p-3 bg-purple-600 hover:bg-purple-700 rounded-lg transition-colors"
          >
            <Wallet size={20} />
            <span>Wallet</span>
          </button>
        </div>

        <div className="space-y-2">
          <h3 className="text-sm font-semibold text-gray-400 uppercase">System Status</h3>
          
          <div className="space-y-1 text-sm">
            <div className="flex justify-between">
              <span>Receiver:</span>
              <span className={`${systemState.components.receiver === 'running' ? 'text-green-400' : 'text-red-400'}`}>
                {systemState.components.receiver || 'offline'}
              </span>
            </div>
            
            <div className="flex justify-between">
              <span>CLI:</span>
              <span className={`${systemState.components.cli === 'running' ? 'text-green-400' : 'text-red-400'}`}>
                {systemState.components.cli || 'offline'}
              </span>
            </div>
            
            <div className="flex justify-between">
              <span>Cognitive:</span>
              <span className={`${systemState.cognitive.hrmActive ? 'text-green-400' : 'text-yellow-400'}`}>
                {systemState.cognitive.hrmActive ? 'active' : 'standby'}
              </span>
            </div>
            
            <div className="flex justify-between">
              <span>Wallet:</span>
              <span className={`${systemState.wallet.connected ? 'text-green-400' : 'text-gray-400'}`}>
                {systemState.wallet.connected ? 'connected' : 'disconnected'}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );

  const renderCLIPanel = () => {
    if (!showCLI) return null;

    return (
      <div className="fixed bottom-0 left-0 right-0 h-64 bg-black border-t border-gray-600 z-50">
        <div className="flex items-center justify-between p-2 bg-gray-800 text-white">
          <span className="text-sm font-semibold">KNIRV Terminal</span>
          <button
            onClick={() => setShowCLI(false)}
            className="text-gray-400 hover:text-white"
          >
            <X size={16} />
          </button>
        </div>
        <iframe
          src="http://localhost:3003"
          className="w-full h-full border-none"
          title="KNIRV CLI Terminal"
        />
      </div>
    );
  };

  return (
    <div className="flex h-screen bg-gray-900">
      {renderDesktopSidebar()}
      {renderReceiverInterface()}
      {renderCLIPanel()}
      
      {/* QR Scanner Modal */}
      {showQRScanner && (
        <div className="fixed inset-0 bg-black bg-opacity-75 flex items-center justify-center z-50">
          <div className="bg-white p-6 rounded-lg max-w-md w-full mx-4">
            <div className="flex justify-between items-center mb-4">
              <h3 className="text-lg font-semibold">QR Code Scanner</h3>
              <button
                onClick={() => setShowQRScanner(false)}
                className="text-gray-500 hover:text-gray-700"
              >
                <X size={20} />
              </button>
            </div>
            <div className="aspect-square bg-gray-200 rounded-lg flex items-center justify-center">
              <p className="text-gray-500">QR Scanner Component</p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default UnifiedInterface;
