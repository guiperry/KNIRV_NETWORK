import { BrowserRouter as Router, Routes, Route } from "react-router";
import DVEList from "@/react-app/pages/DVEList";
import DVECreate from "@/react-app/pages/DVECreate";
import HomePage from "@/react-app/pages/Home";
import Scanner from "@/react-app/pages/Scanner";
import Skills from "@/react-app/pages/Skills";
import UDC from "@/react-app/pages/UDC";
import VaultPage from "@/react-app/pages/VaultPage";
import CognitiveEngineChat from "@/react-app/pages/CognitiveEngineChat";
import AgentChat from "@/react-app/pages/AgentChat";
import Onboarding from "@/react-app/pages/Onboarding";

export default function App() {
  return (
    <Router>
      <Routes>
        <Route path="/" element={<DVEList />} />
        <Route path="/dves" element={<DVEList />} />
        <Route path="/dves/new" element={<DVECreate />} />
        <Route path="/dves/:dveId" element={<DVEList />} />
        <Route path="/dves/:dveId/agent" element={<AgentChat />} />
        <Route path="/vault" element={<VaultPage />} />
        <Route path="/cognitive" element={<CognitiveEngineChat />} />
        <Route path="/workflows" element={<HomePage />} />
        <Route path="/scanner" element={<Scanner />} />
        <Route path="/skills" element={<Skills />} />
        <Route path="/udc" element={<UDC />} />
        <Route path="/onboarding" element={<Onboarding />} />
      </Routes>
    </Router>
  );
}
