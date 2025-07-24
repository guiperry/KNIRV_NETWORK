import React from 'react';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import Layout from './components/Layout';
import Dashboard from './pages/Dashboard';
import Blocks from './pages/Blocks';
import BlockDetails from './pages/BlockDetails';
import Transactions from './pages/Transactions';
import Accounts from './pages/Accounts';
import AccountDetails from './pages/AccountDetails';
import Search from './pages/Search';
import { BlockchainProvider } from './context/BlockchainContext';

function App() {
  return (
    <div className="min-h-screen bg-gray-900 text-white">
      <BlockchainProvider>
        <Router>
          <Layout>
            <Routes>
              <Route path="/" element={<Dashboard />} />
              <Route path="/blocks" element={<Blocks />} />
              <Route path="/block/:height" element={<BlockDetails />} />
              <Route path="/transactions" element={<Transactions />} />
              <Route path="/accounts" element={<Accounts />} />
              <Route path="/account/:address" element={<AccountDetails />} />
              <Route path="/search/:query" element={<Search />} />
            </Routes>
          </Layout>
        </Router>
      </BlockchainProvider>
    </div>
  );
}

export default App;