import React, { useState } from 'react';
import { 
  Calendar,
  BarChart3,
  PieChart,
  LineChart,
  Activity,
  Target,
  Bot,
  Zap,
  GitBranch
} from 'lucide-react';
import { AnalyticsDashboard } from './analytics/AnalyticsDashboard';

export const Analytics = () => {
  const [activeTab, setActiveTab] = useState('dashboard');

  const tabs = [
    { id: 'dashboard', name: 'Dashboard', icon: BarChart3 },
    { id: 'agents', name: 'Agent Analytics', icon: Bot },
    { id: 'targets', name: 'Target Systems', icon: Target },
    { id: 'capabilities', name: 'Capabilities', icon: Zap },
    { id: 'workflows', name: 'Workflows', icon: GitBranch },
  ];

  const renderContent = () => {
    switch (activeTab) {
      case 'dashboard':
        return <AnalyticsDashboard />;
      case 'agents':
        return (
          <div className="p-6">
            <h2 className="text-2xl font-bold text-white mb-4">Agent Analytics</h2>
            <p className="text-slate-400">Detailed analytics for individual agents coming soon.</p>
          </div>
        );
      case 'targets':
        return (
          <div className="p-6">
            <h2 className="text-2xl font-bold text-white mb-4">Target System Analytics</h2>
            <p className="text-slate-400">Detailed analytics for target systems coming soon.</p>
          </div>
        );
      case 'capabilities':
        return (
          <div className="p-6">
            <h2 className="text-2xl font-bold text-white mb-4">Capability Analytics</h2>
            <p className="text-slate-400">Detailed analytics for capabilities coming soon.</p>
          </div>
        );
      case 'workflows':
        return (
          <div className="p-6">
            <h2 className="text-2xl font-bold text-white mb-4">Workflow Analytics</h2>
            <p className="text-slate-400">Detailed analytics for workflows coming soon.</p>
          </div>
        );
      default:
        return <AnalyticsDashboard />;
    }
  };

  return (
    <div>
      {/* Tabs */}
      <div className="bg-slate-800/80 border-b border-slate-700/50">
        <div className="flex overflow-x-auto">
          {tabs.map((tab) => {
            const Icon = tab.icon;
            return (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`flex items-center space-x-2 px-4 py-3 transition-all duration-200 ${
                  activeTab === tab.id
                    ? 'bg-slate-700/50 text-white border-b-2 border-purple-500'
                    : 'text-slate-400 hover:text-white hover:bg-slate-700/30'
                }`}
              >
                <Icon className="w-4 h-4" />
                <span>{tab.name}</span>
              </button>
            );
          })}
        </div>
      </div>

      {/* Content */}
      {renderContent()}
    </div>
  );
};