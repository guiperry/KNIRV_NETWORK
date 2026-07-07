import { BrowserRouter as Router, Routes, Route } from "react-router";
import DVEList from "@/react-app/pages/DVEList";
import DVECreate from "@/react-app/pages/DVECreate";
import HomePage from "@/react-app/pages/Home";
import Scanner from "@/react-app/pages/Scanner";
import Badges from "@/react-app/pages/Badges";
import UDC from "@/react-app/pages/UDC";
import VaultPage from "@/react-app/pages/VaultPage";
import CognitiveEngineChat from "@/react-app/pages/CognitiveEngineChat";
import AgentChat from "@/react-app/pages/AgentChat";
import Onboarding from "@/react-app/pages/Onboarding";
import Settings from "@/react-app/pages/Settings";
import NetworkSelectorDialog from "@/react-app/components/NetworkSelectorDialog";
import { useNetworkConfig } from "@/react-app/hooks/useNetworkConfig";

export default function App() {
  const { networkId, isConfigured, devServerUrl, selectNetwork } = useNetworkConfig();

  if (!isConfigured) {
    return (
      <NetworkSelectorDialog
        isOpen
        currentNetworkId={networkId}
        currentDevServerUrl={devServerUrl}
        onSelect={selectNetwork}
      />
    );
  }

  return (
    <Router>
      <Routes>
        <Route path="/" element={<Onboarding />} />
        <Route path="/onboarding" element={<Onboarding />} />
        <Route path="/dves" element={<DVEList />} />
        <Route path="/dves/new" element={<DVECreate />} />
        <Route path="/dves/:dveId" element={<DVEList />} />
        <Route path="/dves/:dveId/agent" element={<AgentChat />} />
        <Route path="/vault" element={<VaultPage />} />
        <Route path="/cognitive" element={<CognitiveEngineChat />} />
        <Route path="/workflows" element={<HomePage />} />
        <Route path="/scanner" element={<Scanner />} />
        <Route path="/badges" element={<Badges />} />
        <Route path="/udc" element={<UDC />} />
        <Route path="/settings" element={<Settings />} />
      </Routes>
    </Router>
  );
}
