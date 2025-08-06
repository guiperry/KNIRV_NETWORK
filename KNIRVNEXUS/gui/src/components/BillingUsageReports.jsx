import React, { useState, useEffect } from 'react';
import {
  DollarSign,
  TrendingUp,
  Calendar,
  Download,
  Filter,
  CreditCard,
  BarChart3,
  PieChart,
  Clock,
  Cpu,
  Database,
  HardDrive,
  Zap,
  FileText,
  AlertCircle
} from 'lucide-react';

const BillingUsageReports = () => {
  const [selectedPeriod, setSelectedPeriod] = useState('current');
  const [selectedProject, setSelectedProject] = useState('all');
  const [billingData, setBillingData] = useState({});
  const [usageData, setUsageData] = useState([]);
  const [loading, setLoading] = useState(false);

  // Mock billing and usage data
  useEffect(() => {
    const mockBillingData = {
      currentMonth: {
        total: 1247.83,
        breakdown: {
          compute: 687.45,
          storage: 234.12,
          network: 156.78,
          tee: 169.48
        },
        trend: '+12.5%'
      },
      previousMonth: {
        total: 1108.92,
        breakdown: {
          compute: 612.34,
          storage: 198.45,
          network: 142.33,
          tee: 155.80
        }
      },
      yearToDate: 14567.89,
      projectedMonth: 1356.24
    };

    const mockUsageData = [
      {
        id: 'task-001',
        name: 'ML Inference Pipeline',
        project: 'AI Research',
        startTime: new Date(Date.now() - 6 * 60 * 60 * 1000),
        endTime: new Date(Date.now() - 2 * 60 * 60 * 1000),
        duration: '4h 12m',
        resources: {
          cpu: { hours: 4.2, cost: 12.60 },
          memory: { gb: 16, cost: 8.40 },
          storage: { gb: 50, cost: 2.50 },
          tee: { hours: 4.2, cost: 21.00 }
        },
        totalCost: 44.50,
        status: 'completed'
      },
      {
        id: 'task-002',
        name: 'Data Processing Task',
        project: 'Analytics Platform',
        startTime: new Date(Date.now() - 3 * 60 * 60 * 1000),
        endTime: null,
        duration: '3h 15m (running)',
        resources: {
          cpu: { hours: 3.25, cost: 9.75 },
          memory: { gb: 32, cost: 16.80 },
          storage: { gb: 100, cost: 5.00 },
          tee: { hours: 3.25, cost: 16.25 }
        },
        totalCost: 47.80,
        status: 'running'
      },
      {
        id: 'task-003',
        name: 'Security Analysis',
        project: 'Security Audit',
        startTime: new Date(Date.now() - 24 * 60 * 60 * 1000),
        endTime: new Date(Date.now() - 22 * 60 * 60 * 1000),
        duration: '2h 45m',
        resources: {
          cpu: { hours: 2.75, cost: 8.25 },
          memory: { gb: 8, cost: 4.20 },
          storage: { gb: 25, cost: 1.25 },
          tee: { hours: 2.75, cost: 13.75 }
        },
        totalCost: 27.45,
        status: 'completed'
      }
    ];

    setBillingData(mockBillingData);
    setUsageData(mockUsageData);
  }, [selectedPeriod]);

  const formatCurrency = (amount) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD'
    }).format(amount);
  };

  const formatDate = (date) => {
    return date.toLocaleString();
  };

  const getStatusColor = (status) => {
    switch (status) {
      case 'completed':
        return 'text-green-400';
      case 'running':
        return 'text-blue-400';
      case 'failed':
        return 'text-red-400';
      default:
        return 'text-gray-400';
    }
  };

  const CostBreakdownCard = ({ title, amount, percentage, icon: Icon, color }) => (
    <div className="bg-slate-800 rounded-lg p-6 border border-slate-700">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center space-x-3">
          <div className={`p-2 rounded-lg bg-gradient-to-r ${color}`}>
            <Icon className="w-6 h-6 text-white" />
          </div>
          <h3 className="text-lg font-semibold text-white">{title}</h3>
        </div>
        <span className="text-sm text-slate-400">{percentage}%</span>
      </div>
      <p className="text-3xl font-bold text-white">{formatCurrency(amount)}</p>
    </div>
  );

  const filteredUsageData = usageData.filter(item => {
    return selectedProject === 'all' || item.project === selectedProject;
  });

  return (
    <div className="min-h-screen bg-knirv-gradient p-6">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="flex items-center justify-between mb-8">
          <div className="flex items-center space-x-3">
            <DollarSign className="w-8 h-8 text-knirv-primary" />
            <div>
              <h1 className="text-3xl font-bold text-white">Billing & Usage Reports</h1>
              <p className="text-slate-300">Transparent resource consumption and cost breakdown</p>
            </div>
          </div>
          <div className="flex items-center space-x-4">
            <select
              value={selectedPeriod}
              onChange={(e) => setSelectedPeriod(e.target.value)}
              className="bg-slate-800 text-white rounded-lg px-3 py-2 border border-slate-600"
            >
              <option value="current">Current Month</option>
              <option value="previous">Previous Month</option>
              <option value="quarter">This Quarter</option>
              <option value="year">This Year</option>
            </select>
            <button className="flex items-center space-x-2 px-4 py-2 bg-knirv-primary text-white rounded-lg hover:bg-knirv-secondary transition-colors">
              <Download className="w-4 h-4" />
              <span>Export Report</span>
            </button>
          </div>
        </div>

        {/* Billing Summary */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
          <div className="bg-slate-800 rounded-lg p-6 border border-slate-700">
            <div className="flex items-center space-x-3 mb-4">
              <CreditCard className="w-6 h-6 text-knirv-primary" />
              <h3 className="text-lg font-semibold text-white">Current Month</h3>
            </div>
            <p className="text-3xl font-bold text-white">{formatCurrency(billingData.currentMonth?.total || 0)}</p>
            <p className="text-sm text-green-400">{billingData.currentMonth?.trend || '0%'} from last month</p>
          </div>
          <div className="bg-slate-800 rounded-lg p-6 border border-slate-700">
            <div className="flex items-center space-x-3 mb-4">
              <TrendingUp className="w-6 h-6 text-green-400" />
              <h3 className="text-lg font-semibold text-white">Year to Date</h3>
            </div>
            <p className="text-3xl font-bold text-white">{formatCurrency(billingData.yearToDate || 0)}</p>
            <p className="text-sm text-blue-400">Average: {formatCurrency((billingData.yearToDate || 0) / 12)}/month</p>
          </div>
          <div className="bg-slate-800 rounded-lg p-6 border border-slate-700">
            <div className="flex items-center space-x-3 mb-4">
              <Calendar className="w-6 h-6 text-yellow-400" />
              <h3 className="text-lg font-semibold text-white">Projected Month</h3>
            </div>
            <p className="text-3xl font-bold text-white">{formatCurrency(billingData.projectedMonth || 0)}</p>
            <p className="text-sm text-yellow-400">Based on current usage</p>
          </div>
          <div className="bg-slate-800 rounded-lg p-6 border border-slate-700">
            <div className="flex items-center space-x-3 mb-4">
              <BarChart3 className="w-6 h-6 text-purple-400" />
              <h3 className="text-lg font-semibold text-white">Cost Efficiency</h3>
            </div>
            <p className="text-3xl font-bold text-white">94.2%</p>
            <p className="text-sm text-green-400">+2.1% optimization</p>
          </div>
        </div>

        {/* Cost Breakdown */}
        <div className="grid grid-cols-1 lg:grid-cols-4 gap-6 mb-8">
          <CostBreakdownCard
            title="Compute"
            amount={billingData.currentMonth?.breakdown?.compute || 0}
            percentage={55}
            icon={Cpu}
            color="from-blue-500 to-cyan-500"
          />
          <CostBreakdownCard
            title="Storage"
            amount={billingData.currentMonth?.breakdown?.storage || 0}
            percentage={19}
            icon={HardDrive}
            color="from-green-500 to-emerald-500"
          />
          <CostBreakdownCard
            title="Network"
            amount={billingData.currentMonth?.breakdown?.network || 0}
            percentage={13}
            icon={Zap}
            color="from-yellow-500 to-orange-500"
          />
          <CostBreakdownCard
            title="TEE Premium"
            amount={billingData.currentMonth?.breakdown?.tee || 0}
            percentage={13}
            icon={Database}
            color="from-knirv-primary to-knirv-secondary"
          />
        </div>

        {/* Usage Details */}
        <div className="bg-slate-800 rounded-lg border border-slate-700">
          <div className="p-6 border-b border-slate-700">
            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-3">
                <FileText className="w-6 h-6 text-knirv-primary" />
                <h2 className="text-xl font-semibold text-white">Detailed Usage Report</h2>
              </div>
              <div className="flex items-center space-x-4">
                <select
                  value={selectedProject}
                  onChange={(e) => setSelectedProject(e.target.value)}
                  className="bg-slate-700 text-white rounded-lg px-3 py-2 border border-slate-600"
                >
                  <option value="all">All Projects</option>
                  <option value="AI Research">AI Research</option>
                  <option value="Analytics Platform">Analytics Platform</option>
                  <option value="Security Audit">Security Audit</option>
                </select>
              </div>
            </div>
          </div>
          <div className="p-6">
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-slate-700">
                    <th className="text-left py-3 px-4 text-slate-300 font-medium">Task</th>
                    <th className="text-left py-3 px-4 text-slate-300 font-medium">Project</th>
                    <th className="text-left py-3 px-4 text-slate-300 font-medium">Duration</th>
                    <th className="text-left py-3 px-4 text-slate-300 font-medium">CPU</th>
                    <th className="text-left py-3 px-4 text-slate-300 font-medium">Memory</th>
                    <th className="text-left py-3 px-4 text-slate-300 font-medium">Storage</th>
                    <th className="text-left py-3 px-4 text-slate-300 font-medium">TEE</th>
                    <th className="text-left py-3 px-4 text-slate-300 font-medium">Total Cost</th>
                    <th className="text-left py-3 px-4 text-slate-300 font-medium">Status</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredUsageData.map((item) => (
                    <tr key={item.id} className="border-b border-slate-700/50 hover:bg-slate-700/30">
                      <td className="py-3 px-4">
                        <div>
                          <p className="text-white font-medium">{item.name}</p>
                          <p className="text-xs text-slate-400">{item.id}</p>
                        </div>
                      </td>
                      <td className="py-3 px-4 text-slate-300">{item.project}</td>
                      <td className="py-3 px-4 text-slate-300">{item.duration}</td>
                      <td className="py-3 px-4 text-white">{formatCurrency(item.resources.cpu.cost)}</td>
                      <td className="py-3 px-4 text-white">{formatCurrency(item.resources.memory.cost)}</td>
                      <td className="py-3 px-4 text-white">{formatCurrency(item.resources.storage.cost)}</td>
                      <td className="py-3 px-4 text-white">{formatCurrency(item.resources.tee.cost)}</td>
                      <td className="py-3 px-4 text-white font-semibold">{formatCurrency(item.totalCost)}</td>
                      <td className="py-3 px-4">
                        <span className={`capitalize ${getStatusColor(item.status)}`}>
                          {item.status}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>

        {/* Cost Optimization Tips */}
        <div className="mt-8 bg-slate-800 rounded-lg border border-slate-700 p-6">
          <div className="flex items-center space-x-3 mb-4">
            <AlertCircle className="w-6 h-6 text-yellow-400" />
            <h3 className="text-lg font-semibold text-white">Cost Optimization Recommendations</h3>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="bg-slate-700 rounded-lg p-4">
              <h4 className="text-white font-medium mb-2">Right-size Resources</h4>
              <p className="text-sm text-slate-300">
                Task "ML Inference Pipeline" is using 50% more CPU than needed. Consider reducing allocation to save ~$15/month.
              </p>
            </div>
            <div className="bg-slate-700 rounded-lg p-4">
              <h4 className="text-white font-medium mb-2">Optimize Storage</h4>
              <p className="text-sm text-slate-300">
                Clean up unused storage in "Analytics Platform" project to reduce costs by ~$8/month.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default BillingUsageReports;
