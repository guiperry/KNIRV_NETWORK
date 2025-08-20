import { BrowserRouter as Router, Routes, Route } from "react-router";
import HomePage from "@/react-app/pages/Home";
import Skills from "@/react-app/pages/Skills";
import UDC from "@/react-app/pages/UDC";
import WalletPage from "@/react-app/pages/Wallet";

export default function App() {
  return (
    <Router>
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/skills" element={<Skills />} />
        <Route path="/udc" element={<UDC />} />
        <Route path="/wallet" element={<WalletPage />} />
      </Routes>
    </Router>
  );
}
