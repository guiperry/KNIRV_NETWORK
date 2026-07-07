import React, { useState } from 'react';
import { Cpu, Zap, Shield, MapPin, Activity, ChevronRight, Plus, MessageSquare, BadgeCheck, Wifi, WifiOff } from 'lucide-react';
import Layout from '@/react-app/components/Layout';

// ─── Types ───────────────────────────────────────────────────────────────────

type DVEStatus = 'online' | 'degraded' | 'offline';
type TEEType = 'sgx' | 'browser-ext' | 'tdx';

interface Badge {
  label: string;
  color: 'blue' | 'green' | 'purple' | 'amber' | 'cyan';
}

interface DVEData {
  id: string;
  name: string;
  status: DVEStatus;
  tee_type: TEEType;
  vault: string;
  nrn_staked: number;
  reputation: number;
  location: string;
  badges: Badge[];
  cpu: number;
  memory: number;
  uptime: string;
  version: string;
}

interface NetworkDVE {
  id: string;
  name: string;
  tee_type: string;
  location: string;
  reputation: number;
  is_online: boolean;
}

// ─── Mock Data ───────────────────────────────────────────────────────────────

const myDVEs: DVEData[] = [
  {
    id: 'DVE-ALPHA-7X',
    name: 'DVE-Alpha',
    status: 'online',
    tee_type: 'sgx',
    vault: '0x7a3b...c9f2',
    nrn_staked: 12000,
    reputation: 847,
    location: 'us-east-1',
    badges: [
      { label: 'ATTESTED', color: 'green' },
      { label: 'SGX 2.0', color: 'blue' },
      { label: 'LOW LATENCY', color: 'cyan' },
    ],
    cpu: 34,
    memory: 52,
    uptime: '14d 7h 32m',
    version: 'v2.1.0',
  },
  {
    id: 'DVE-BETA-3Y',
    name: 'DVE-Beta',
    status: 'degraded',
    tee_type: 'browser-ext',
    vault: '0xd45f...ab81',
    nrn_staked: 1000,
    reputation: 203,
    location: 'local',
    badges: [
      { label: 'BROWSER', color: 'purple' },
      { label: 'DEV MODE', color: 'amber' },
    ],
    cpu: 12,
    memory: 78,
    uptime: '2d 3h 15m',
    version: 'v0.9.4',
  },
  {
    id: 'DVE-GAMMA-1Z',
    name: 'DVE-Gamma',
    status: 'online',
    tee_type: 'tdx',
    vault: '0xef8c...44d0',
    nrn_staked: 5000,
    reputation: 512,
    location: 'eu-central',
    badges: [
      { label: 'TDX', color: 'blue' },
      { label: 'CONFIDENTIAL', color: 'purple' },
      { label: 'VERIFIED', color: 'green' },
    ],
    cpu: 56,
    memory: 44,
    uptime: '7d 12h 08m',
    version: 'v2.0.3',
  },
];

const networkDVEs: NetworkDVE[] = [
  { id: 'NODE-9F2A', name: 'DVE-Delta', tee_type: 'SGX', location: 'ap-southeast-1', reputation: 921, is_online: true },
  { id: 'NODE-3B7C', name: 'DVE-Epsilon', tee_type: 'TDX', location: 'eu-west-2', reputation: 688, is_online: true },
  { id: 'NODE-A1D4', name: 'DVE-Zeta', tee_type: 'SEV-SNP', location: 'us-west-2', reputation: 774, is_online: false },
  { id: 'NODE-8E5F', name: 'DVE-Eta', tee_type: 'SGX', location: 'ap-northeast-1', reputation: 445, is_online: true },
  { id: 'NODE-2C9B', name: 'DVE-Theta', tee_type: 'browser-ext', location: 'sa-east-1', reputation: 312, is_online: false },
];

// ─── Inline Sub-Components ──────────────────────────────────────────────────

const StatusDot: React.FC<{ status: DVEStatus }> = ({ status }) => {
  const colors: Record<DVEStatus, string> = {
    online: 'bg-green-500 shadow-[0_0_8px_rgba(34,197,94,0.6)]',
    degraded: 'bg-yellow-500 shadow-[0_0_8px_rgba(234,179,8,0.6)]',
    offline: 'bg-red-500 shadow-[0_0_8px_rgba(239,68,68,0.6)]',
  };
  return <span className={`inline-block w-2.5 h-2.5 rounded-full ${colors[status]} animate-pulse`} />;
};

const DVEStatusBadge: React.FC<{ status: DVEStatus }> = ({ status }) => {
  const config: Record<DVEStatus, { label: string; classes: string }> = {
    online: { label: 'ONLINE', classes: 'bg-green-500/20 text-green-400 border-green-500/30' },
    degraded: { label: 'DEGRADED', classes: 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30' },
    offline: { label: 'OFFLINE', classes: 'bg-red-500/20 text-red-400 border-red-500/30' },
  };
  const { label, classes } = config[status];
  return (
    <span className={`inline-flex items-center rounded-full border px-2 py-0.5 text-[9px] font-black uppercase tracking-wider ${classes}`}>
      {label}
    </span>
  );
};

const BadgeChip: React.FC<{ badge: Badge }> = ({ badge }) => {
  const colorMap: Record<string, string> = {
    blue: 'bg-blue-500/15 text-blue-400 border-blue-500/25',
    green: 'bg-green-500/15 text-green-400 border-green-500/25',
    purple: 'bg-purple-500/15 text-purple-400 border-purple-500/25',
    amber: 'bg-amber-500/15 text-amber-400 border-amber-500/25',
    cyan: 'bg-cyan-500/15 text-cyan-400 border-cyan-500/25',
  };
  return (
    <span className={`inline-flex items-center rounded-md border px-1.5 py-0.5 text-[8px] font-black uppercase tracking-wider ${colorMap[badge.color] || 'bg-slate-500/15 text-slate-400 border-slate-500/25'}`}>
      <BadgeCheck className="w-2.5 h-2.5 mr-0.5" />
      {badge.label}
    </span>
  );
};

const TEEChip: React.FC<{ type: TEEType }> = ({ type }) => {
  const config: Record<TEEType, { icon: React.ReactNode; label: string }> = {
    sgx: { icon: <Shield className="w-3.5 h-3.5 text-blue-400" />, label: 'SGX' },
    'browser-ext': { icon: <Cpu className="w-3.5 h-3.5 text-purple-400" />, label: 'BROWSER' },
    tdx: { icon: <Shield className="w-3.5 h-3.5 text-cyan-400" />, label: 'TDX' },
  };
  const { icon, label } = config[type];
  return (
    <div className="flex items-center space-x-1.5 px-2 py-1 bg-slate-800/80 rounded-lg border border-slate-700/50">
      {icon}
      <span className="text-[10px] font-bold text-slate-300">{label}</span>
    </div>
  );
};

const ProgressBar: React.FC<{ value: number; className?: string }> = ({ value, className }) => (
  <div className={`relative h-1.5 w-full overflow-hidden rounded-full bg-slate-800 ${className || ''}`}>
    <div
      className="h-full rounded-full bg-gradient-to-r from-blue-500 to-blue-700 transition-all duration-500"
      style={{ width: `${Math.min(value, 100)}%` }}
    />
  </div>
);

const ActionButton: React.FC<{ children: React.ReactNode; onClick?: () => void; variant?: 'primary' | 'outline' | 'ghost'; className?: string }> = ({ children, onClick, variant = 'primary', className }) => {
  const base = 'inline-flex items-center justify-center rounded-lg text-xs font-bold uppercase tracking-wider transition-all duration-200 focus:outline-none';
  const variants = {
    primary: 'bg-blue-600 text-white hover:bg-blue-500 shadow-lg shadow-blue-600/20',
    outline: 'border border-blue-600/30 text-blue-400 hover:bg-blue-600 hover:text-white',
    ghost: 'text-slate-400 hover:text-white hover:bg-white/5',
  };
  return (
    <button onClick={onClick} className={`${base} ${variants[variant]} ${className || 'px-3 py-1.5'}`}>
      {children}
    </button>
  );
};

// ─── DVE Card ────────────────────────────────────────────────────────────────

const DVECard: React.FC<{ dve: DVEData }> = ({ dve }) => {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="relative group">
      <div className={`absolute -inset-0.5 bg-gradient-to-r from-blue-600/30 to-blue-800/30 rounded-2xl blur opacity-20 group-hover:opacity-50 transition duration-300`} />

      <div className="relative glass-panel overflow-hidden glow-hover">
        {/* Card Header */}
        <div
          className="p-4 cursor-pointer select-none"
          onClick={() => setExpanded(!expanded)}
        >
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3 min-w-0">
              <div className={`flex-shrink-0 w-10 h-10 rounded-xl flex items-center justify-center border ${
                dve.status === 'online' ? 'bg-green-500/10 border-green-500/20' :
                dve.status === 'degraded' ? 'bg-yellow-500/10 border-yellow-500/20' :
                'bg-red-500/10 border-red-500/20'
              }`}>
                {dve.status === 'online' ? (
                  <Wifi className="w-5 h-5 text-green-400" />
                ) : dve.status === 'degraded' ? (
                  <Wifi className="w-5 h-5 text-yellow-400" />
                ) : (
                  <WifiOff className="w-5 h-5 text-red-400" />
                )}
              </div>
              <div className="min-w-0">
                <div className="flex items-center space-x-2">
                  <h3 className="text-sm font-bold text-white truncate">{dve.name}</h3>
                  <StatusDot status={dve.status} />
                </div>
                <div className="flex items-center space-x-2 mt-0.5">
                  <span className="text-[10px] font-mono text-slate-500">{dve.id}</span>
                  <span className="text-slate-600">•</span>
                  <span className="text-[10px] font-mono text-slate-500 truncate">{dve.vault}</span>
                </div>
              </div>
            </div>

            <div className="flex items-center space-x-2 flex-shrink-0">
              <DVEStatusBadge status={dve.status} />
              <ChevronRight className={`w-4 h-4 text-slate-500 transition-transform duration-200 ${expanded ? 'rotate-90' : ''}`} />
            </div>
          </div>

          {/* Quick Stats Row */}
          <div className="flex items-center space-x-4 mt-3">
            <TEEChip type={dve.tee_type} />
            <div className="flex items-center space-x-1 text-[10px] text-slate-400 font-mono">
              <MapPin className="w-3 h-3 text-red-400" />
              <span>{dve.location}</span>
            </div>
            <div className="flex items-center space-x-1 text-[10px] text-slate-400 font-mono ml-auto">
              <Zap className="w-3 h-3 text-amber-400" />
              <span className="font-bold text-amber-300">{dve.nrn_staked.toLocaleString()} NRN</span>
            </div>
          </div>

          {/* Badges Row */}
          <div className="flex flex-wrap items-center gap-1.5 mt-2">
            {dve.badges.map((badge, i) => (
              <BadgeChip key={i} badge={badge} />
            ))}
          </div>

          {/* Reputation + Chat Row */}
          <div className="flex items-center justify-between mt-3">
            <div className="flex items-center space-x-2">
              <Activity className="w-3.5 h-3.5 text-blue-400" />
              <span className="text-[11px] font-bold font-mono text-blue-300">
                Rep: {dve.reputation}
                <span className="text-slate-600 font-normal">/1000</span>
              </span>
            </div>
            <ActionButton variant="outline" className="h-7 px-2.5 text-[9px] space-x-1">
              <MessageSquare className="w-3 h-3" />
              <span>Chat ▶</span>
            </ActionButton>
          </div>
        </div>

        {/* Expandable Detail Panel */}
        {expanded && (
          <div className="border-t border-white/5 animate-[slideDown_0.2s_ease-out]">
            <div className="p-4 space-y-4 bg-slate-900/30">
              {/* Metrics */}
              <div className="grid grid-cols-2 gap-3">
                <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-3 space-y-1.5">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center text-[9px] font-black text-slate-400 uppercase tracking-wider">
                      <Cpu className="w-3 h-3 mr-1.5 text-blue-500" />
                      CPU
                    </div>
                    <span className="text-[10px] font-mono font-bold text-blue-300">{dve.cpu}%</span>
                  </div>
                  <ProgressBar value={dve.cpu} />
                </div>
                <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-3 space-y-1.5">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center text-[9px] font-black text-slate-400 uppercase tracking-wider">
                      <Zap className="w-3 h-3 mr-1.5 text-purple-500" />
                      MEM
                    </div>
                    <span className="text-[10px] font-mono font-bold text-purple-300">{dve.memory}%</span>
                  </div>
                  <ProgressBar value={dve.memory} />
                </div>
              </div>

              {/* Details Grid */}
              <div className="grid grid-cols-2 gap-2">
                <DetailItem label="Uptime" value={dve.uptime} />
                <DetailItem label="Version" value={dve.version} />
                <DetailItem label="TEE Type" value={dve.tee_type.toUpperCase()} />
                <DetailItem label="Vault" value={dve.vault} />
              </div>

              {/* Actions */}
              <div className="flex space-x-2">
                <ActionButton variant="outline" className="flex-1 h-8 text-[9px]">
                  View Details
                </ActionButton>
                <ActionButton variant="primary" className="flex-1 h-8 text-[9px] space-x-1">
                  <MessageSquare className="w-3 h-3" />
                  <span>Chat ▶</span>
                </ActionButton>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

const DetailItem: React.FC<{ label: string; value: string }> = ({ label, value }) => (
  <div className="bg-slate-950 border border-slate-800 rounded-lg p-2.5">
    <p className="text-[8px] font-black text-slate-500 uppercase tracking-widest">{label}</p>
    <p className="text-[10px] font-bold text-slate-200 font-mono mt-0.5 truncate">{value}</p>
  </div>
);

// ─── Network DVE Row ─────────────────────────────────────────────────────────

const NetworkDVERow: React.FC<{ node: NetworkDVE }> = ({ node }) => (
  <div className="flex items-center justify-between p-3 bg-slate-900/40 hover:bg-slate-900/70 transition-colors border border-slate-800 rounded-xl group">
    <div className="flex items-center space-x-3 min-w-0">
      <div className={`flex-shrink-0 w-8 h-8 rounded-lg flex items-center justify-center border ${
        node.is_online ? 'bg-green-500/10 border-green-500/20' : 'bg-red-500/10 border-red-500/20'
      }`}>
        {node.is_online ? (
          <Wifi className="w-4 h-4 text-green-400" />
        ) : (
          <WifiOff className="w-4 h-4 text-red-400" />
        )}
      </div>
      <div className="min-w-0">
        <div className="flex items-center space-x-2">
          <span className="text-xs font-bold text-white truncate">{node.name}</span>
          <span className={`inline-block w-1.5 h-1.5 rounded-full ${node.is_online ? 'bg-green-500' : 'bg-red-500'}`} />
        </div>
        <div className="flex items-center space-x-2 text-[9px] font-mono text-slate-500 mt-0.5">
          <span>{node.id}</span>
          <span>•</span>
          <span>{node.tee_type}</span>
          <span>•</span>
          <span><MapPin className="w-2.5 h-2.5 inline text-red-500" /> {node.location}</span>
        </div>
      </div>
    </div>
    <div className="flex items-center space-x-3 flex-shrink-0">
      <span className="text-[10px] font-mono font-bold text-blue-300">Rep: {node.reputation}</span>
      <ChevronRight className="w-3.5 h-3.5 text-slate-600 group-hover:text-slate-400 transition-colors" />
    </div>
  </div>
);

// ─── Network DVEs Section ────────────────────────────────────────────────────

const NetworkDVEsSection: React.FC = () => {
  const [collapsed, setCollapsed] = useState(true);

  return (
    <div className="relative group">
      <div className="absolute -inset-0.5 bg-gradient-to-r from-blue-600/20 to-purple-800/20 rounded-2xl blur opacity-20" />
      <div className="relative glass-panel overflow-hidden glow-hover">
        {/* Section Header */}
        <button
          onClick={() => setCollapsed(!collapsed)}
          className="w-full flex items-center justify-between p-4 cursor-pointer select-none"
        >
          <div className="flex items-center space-x-3">
            <div className="w-9 h-9 bg-purple-500/20 rounded-xl flex items-center justify-center border border-purple-500/20">
              <Wifi className="w-4 h-4 text-purple-400" />
            </div>
            <div className="text-left">
              <h3 className="text-sm font-bold text-white">NETWORK DVEs</h3>
              <p className="text-[9px] font-mono text-slate-500 uppercase tracking-wider">
                Discoverable Nodes • <span className="text-green-400">{networkDVEs.filter(n => n.is_online).length} online</span> • {networkDVEs.length} total
              </p>
            </div>
          </div>
          <ChevronRight className={`w-4 h-4 text-slate-500 transition-transform duration-200 ${!collapsed ? 'rotate-90' : ''}`} />
        </button>

        {/* Collapsible List */}
        {!collapsed && (
          <div className="px-4 pb-4 space-y-2 border-t border-white/5 pt-3 animate-[slideDown_0.2s_ease-out]">
            <div className="text-[9px] font-mono text-slate-500 px-1 pb-1 flex items-center space-x-2">
              <MapPin className="w-3 h-3 text-red-400" />
              <span>DISCOVERABLE NODES ACROSS D-TEN NETWORK</span>
            </div>
            {networkDVEs.map((node) => (
              <NetworkDVERow key={node.id} node={node} />
            ))}
            <button className="w-full mt-2 py-2.5 border border-dashed border-slate-700 hover:border-blue-500/40 rounded-xl text-[10px] font-bold text-slate-500 hover:text-blue-400 transition-all flex items-center justify-center space-x-1.5">
              <Plus className="w-3.5 h-3.5" />
              <span>Discover More Nodes</span>
            </button>
          </div>
        )}
      </div>
    </div>
  );
};

// ─── Main Component ──────────────────────────────────────────────────────────

const DVEList: React.FC = () => {
  return (
    <Layout>
      <div className="p-4 pb-24 space-y-6 relative max-w-2xl mx-auto">
        {/* Header */}
        <div className="text-center py-4 relative">
          <div className="flex items-center justify-center space-x-2 mb-1">
            <div className="w-6 h-6 bg-blue-600 rounded-sm transform rotate-45"></div>
            <h2 className="text-2xl font-extrabold tracking-tighter uppercase text-white">
              KNIRV<span className="text-blue-500 font-light">CONTROLLER</span>
            </h2>
          </div>
          <p className="text-slate-400 text-sm font-mono uppercase tracking-wider">
            Distributed Verification Environment
          </p>
        </div>

        {/* Vault NRN Balance Card */}
        <div className="relative group">
          <div className="absolute -inset-0.5 bg-gradient-to-r from-blue-600/50 to-blue-800/50 rounded-2xl blur opacity-30 group-hover:opacity-50 transition duration-300" />
          <div className="relative glass-panel p-4 glow-hover flex items-center justify-between">
            <div className="flex items-center space-x-3">
              <div className="w-10 h-10 bg-blue-600 rounded-xl flex items-center justify-center">
                <Zap className="w-5 h-5 text-white" />
              </div>
              <div>
                <p className="text-[9px] font-black text-slate-400 uppercase tracking-widest">Vault Balance</p>
                <p className="text-lg font-bold text-white font-mono">
                  18,247 <span className="text-blue-400 text-sm">NRN</span>
                </p>
              </div>
            </div>
            <div className="flex items-center space-x-2">
              <span className="px-2 py-1 rounded-full bg-green-500/20 border border-green-500/30 text-[9px] font-black text-green-400 uppercase tracking-wider">
                +2.4% Today
              </span>
            </div>
          </div>
        </div>

        {/* MY DVEs Section */}
        <div>
          <div className="flex items-center justify-between mb-3">
            <div className="flex items-center space-x-2">
              <Shield className="w-4 h-4 text-blue-400" />
              <h3 className="text-sm font-bold text-white uppercase tracking-wider">My DVEs</h3>
            </div>
            <span className="text-[10px] font-mono text-slate-500 bg-slate-800/60 px-2 py-0.5 rounded-full border border-slate-700/50">
              {myDVEs.length} nodes
            </span>
          </div>

          <div className="space-y-3">
            {myDVEs.map((dve) => (
              <DVECard key={dve.id} dve={dve} />
            ))}
          </div>

          {/* Add DVE Button */}
          <button className="relative group w-full mt-3">
            <div className="absolute -inset-0.5 bg-gradient-to-r from-blue-600/30 to-blue-800/30 rounded-2xl blur opacity-20 group-hover:opacity-50 transition duration-300" />
            <div className="relative glass-panel p-3 glow-hover flex items-center justify-center space-x-2 border-dashed border-blue-500/20 hover:border-blue-500/50 transition-all">
              <Plus className="w-4 h-4 text-blue-400" />
              <span className="text-xs font-bold text-blue-400 uppercase tracking-wider">Add New DVE</span>
            </div>
          </button>
        </div>

        {/* NETWORK DVEs Section */}
        <NetworkDVEsSection />
      </div>
    </Layout>
  );
};

export default DVEList;
