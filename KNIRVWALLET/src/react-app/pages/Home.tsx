import { Bot, Zap, Shield, Activity, TrendingUp, Clock } from 'lucide-react';
import Layout from '@/react-app/components/Layout';
import StatsCard from '@/react-app/components/StatsCard';
import WorkflowCard from '@/react-app/components/WorkflowCard';

export default function Home() {
  const workflows = [
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
        {/* Welcome Section - Gradient Text */}
        <div className="text-center py-6">
          <h2 className="text-2xl font-bold gradient-text mb-2">
            Autonomous Gateway Active
          </h2>
          <p className="text-slate-400 text-sm font-mono uppercase tracking-wider">
            Your AI workflows are connected to the D-TEN network
          </p>
        </div>

        {/* Stats Grid - Glass Panel Style */}
        <div className="grid grid-cols-2 gap-4">
          <StatsCard
            title="Active Workflows"
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

        {/* SEAL Loop Status - Blue Accent */}
        <div className="relative group">
          <div className="absolute -inset-0.5 bg-gradient-to-r from-blue-600/50 to-blue-800/50 rounded-xl blur opacity-30 group-hover:opacity-60 transition duration-300"></div>
          <div className="relative glass-panel p-4 glow-hover">
            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-3">
                <div className="w-3 h-3 bg-blue-400 rounded-full animate-pulse pulse-blue"></div>
                <div>
                  <h3 className="font-semibold text-white">SEAL Loop Active</h3>
                  <p className="text-sm text-blue-400 font-mono">Continuous optimization running</p>
                </div>
              </div>
              <div className="text-right">
                <p className="text-sm text-slate-400 font-mono">Next cycle</p>
                <p className="text-xs text-slate-500 font-mono">in 3m 24s</p>
              </div>
            </div>
          </div>
        </div>

        {/* Workflows Section */}
        <div>
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-white">Your Workflows</h3>
            <button className="text-sm text-blue-400 hover:text-blue-300 transition-colors font-mono uppercase">
              View All
            </button>
          </div>
          
          <div className="space-y-4">
            {workflows.map((workflow) => (
              <WorkflowCard key={workflow.name} {...workflow} />
            ))}
          </div>
        </div>

        {/* Quick Actions */}
        <div>
          <h3 className="text-lg font-semibold text-white mb-4">Quick Actions</h3>
          <div className="grid grid-cols-2 gap-3">
            <ActionButton
              icon={Bot}
              title="Deploy Workflow"
              description="Launch new AI workflow"
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
      <div className="absolute -inset-0.5 bg-gradient-to-r from-blue-600/30 to-blue-800/30 rounded-xl blur opacity-20 group-hover:opacity-50 group-active:opacity-75 transition duration-200"></div>
      <div className="relative glass-panel p-4 hover:border-blue-500/50 group-active:scale-95 transition-all text-left glow-hover">
        <div className="flex items-center space-x-3">
          <div className="w-10 h-10 bg-blue-500/20 rounded-lg flex items-center justify-center border border-blue-500/20">
            <Icon className="w-5 h-5 text-blue-400" />
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
