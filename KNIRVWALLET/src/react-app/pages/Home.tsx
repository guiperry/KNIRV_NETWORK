import { Bot, Zap, Shield, Activity, TrendingUp, Clock } from 'lucide-react';
import Layout from '@/react-app/components/Layout';
import StatsCard from '@/react-app/components/StatsCard';
import AgentCard from '@/react-app/components/AgentCard';

export default function Home() {
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
    <Layout>
      <div className="p-4 pb-24 space-y-6">
        {/* Welcome Section */}
        <div className="text-center py-6">
          <h2 className="text-2xl font-bold bg-gradient-to-r from-purple-400 to-cyan-400 bg-clip-text text-transparent mb-2">
            Autonomous Gateway Active
          </h2>
          <p className="text-slate-400 text-sm">
            Your AI agents are connected to the D-TEN network
          </p>
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
            icon={Zap}
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
            />
            <ActionButton
              icon={TrendingUp}
              title="View Analytics"
              description="Performance insights"
            />
            <ActionButton
              icon={Clock}
              title="Schedule Task"
              description="Automate workflows"
            />
            <ActionButton
              icon={Shield}
              title="Renew UDC"
              description="Extend certificate"
            />
          </div>
        </div>
      </div>
    </Layout>
  );
}

interface ActionButtonProps {
  icon: React.ComponentType<{ className?: string }>;
  title: string;
  description: string;
}

function ActionButton({ icon: Icon, title, description }: ActionButtonProps) {
  return (
    <button className="relative group">
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
