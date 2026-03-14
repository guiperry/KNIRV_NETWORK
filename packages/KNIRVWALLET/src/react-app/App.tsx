import { BrowserRouter as Router, Routes, Route } from "react-router";
import DVENodeVerifier from "@/react-app/pages/DVENodeVerifier";
import HomePage from "@/react-app/pages/Home";
import Scanner from "@/react-app/pages/Scanner";
import Skills from "@/react-app/pages/Skills";
import UDC from "@/react-app/pages/UDC";
import WalletPage from "@/react-app/pages/Wallet";
import Onboarding from "@/react-app/pages/Onboarding";

export default function App() {
  return (
    <Router>
      <Routes>
        <Route path="/" element={<DVENodeVerifier />} />
        <Route path="/workflows" element={<HomePage />} />
        <Route path="/scanner" element={<Scanner />} />
        <Route path="/skills" element={<Skills />} />
        <Route path="/udc" element={<UDC />} />
        <Route path="/wallet" element={<WalletPage />} />
        <Route path="/onboarding" element={<Onboarding />} />
      </Routes>
    </Router>
  );
}
