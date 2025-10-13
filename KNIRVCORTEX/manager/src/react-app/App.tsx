import { BrowserRouter as Router, Routes, Route } from "react-router";
import { ComponentBridge } from "@/shared/ComponentBridge";
import { useEffect, useState } from "react";
import UnifiedInterface from "@/react-app/components/UnifiedInterface";
import Skills from "@/react-app/pages/Skills";
import UDC from "@/react-app/pages/UDC";
import WalletPage from "@/react-app/pages/Wallet";

export default function App() {
  const [bridge, setBridge] = useState<ComponentBridge | null>(null);

  useEffect(() => {
    // Initialize component bridge for manager
    const componentBridge = new ComponentBridge({
      name: 'manager',
      port: 3001,
      endpoints: {
        health: '/health',
        api: '/api',
        wallet: '/wallet',
        qr: '/qr'
      },
      features: {
        qrScanning: true,
        walletIntegration: true,
        voiceControl: true,
        mobileOptimized: true
      }
    });

    setBridge(componentBridge);

    return () => {
      componentBridge.disconnect();
    };
  }, []);

  if (!bridge) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-gray-900 text-white">
        <div className="text-center">
          <div className="animate-spin rounded-full h-32 w-32 border-b-2 border-blue-500 mx-auto mb-4"></div>
          <p>Initializing KNIRV Controller...</p>
        </div>
      </div>
    );
  }

  return (
    <Router>
      <Routes>
        <Route path="/" element={<UnifiedInterface bridge={bridge} />} />
        <Route path="/skills" element={<Skills />} />
        <Route path="/udc" element={<UDC />} />
        <Route path="/wallet" element={<WalletPage />} />
      </Routes>
    </Router>
  );
}
