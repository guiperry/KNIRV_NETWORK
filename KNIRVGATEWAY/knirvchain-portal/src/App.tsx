import React from 'react';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import Layout from './components/Layout';
import Dashboard from './pages/Dashboard';
import SkillNodes from './pages/SkillNodes';
import SkillNodeDetails from './pages/SkillNodeDetails';
import ErrorNodes from './pages/ErrorNodes';
import ErrorNodeDetails from './pages/ErrorNodeDetails';
import VectorDetails from './pages/VectorDetails';
import GraphVisualization from './pages/GraphVisualization';
import Search from './pages/Search';
import { GraphChainProvider } from './context/GraphChainContext';

function App() {
  return (
    <div className="min-h-screen bg-gray-900 text-white">
      <GraphChainProvider>
        <Router>
          <Layout>
            <Routes>
              <Route path="/" element={<Dashboard />} />
              <Route path="/skills" element={<SkillNodes />} />
              <Route path="/skill/:id" element={<SkillNodeDetails />} />
              <Route path="/errors" element={<ErrorNodes />} />
              <Route path="/error/:id" element={<ErrorNodeDetails />} />
              <Route path="/vector/:id" element={<VectorDetails />} />
              <Route path="/graph" element={<GraphVisualization />} />
              <Route path="/search/:query" element={<Search />} />
            </Routes>
          </Layout>
        </Router>
      </GraphChainProvider>
    </div>
  );
}

export default App;
