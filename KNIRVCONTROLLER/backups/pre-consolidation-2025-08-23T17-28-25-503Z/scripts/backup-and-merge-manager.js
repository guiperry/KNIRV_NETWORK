#!/usr/bin/env node

/**
 * KNIRV Controller Manager-Receiver Merge Script
 * 
 * This script backs up and merges the manager application into the receiver,
 * creating a unified application with navigation between both interfaces.
 * 
 * Features:
 * - Creates timestamped backups of both applications
 * - Merges manager components and pages into receiver
 * - Updates receiver App.tsx with routing to manager functionality
 * - Merges dependencies from both package.json files
 * - Adds navigation buttons between interfaces
 * - Preserves all existing functionality
 */

import fs from 'fs/promises';
import path from 'path';
import { execSync } from 'child_process';

const SCRIPT_DIR = process.cwd();
const MANAGER_DIR = path.join(SCRIPT_DIR, 'manager');
const RECEIVER_DIR = path.join(SCRIPT_DIR, 'receiver');
const BACKUP_DIR = path.join(SCRIPT_DIR, 'backups');
const ROOT_FRONTEND_DIR = path.join(SCRIPT_DIR, 'frontend');
const BACKEND_DIR = path.join(SCRIPT_DIR, 'backend');

// Utility functions
const log = (message, type = 'info') => {
  const timestamp = new Date().toISOString();
  const prefix = type === 'error' ? '❌' : type === 'success' ? '✅' : 'ℹ️';
  console.log(`${prefix} [${timestamp}] ${message}`);
};

const ensureDir = async (dirPath) => {
  try {
    await fs.access(dirPath);
  } catch {
    await fs.mkdir(dirPath, { recursive: true });
    log(`Created directory: ${dirPath}`);
  }
};

const copyDirectory = async (src, dest) => {
  await ensureDir(dest);
  const entries = await fs.readdir(src, { withFileTypes: true });
  
  for (const entry of entries) {
    const srcPath = path.join(src, entry.name);
    const destPath = path.join(dest, entry.name);
    
    if (entry.isDirectory()) {
      // Skip node_modules and dist directories
      if (['node_modules', 'dist', '.git'].includes(entry.name)) {
        continue;
      }
      await copyDirectory(srcPath, destPath);
    } else {
      await fs.copyFile(srcPath, destPath);
    }
  }
};

const readJsonFile = async (filePath) => {
  const content = await fs.readFile(filePath, 'utf8');
  return JSON.parse(content);
};

const writeJsonFile = async (filePath, data) => {
  await fs.writeFile(filePath, JSON.stringify(data, null, 2) + '\n');
};

// Main backup function
const createBackups = async () => {
  log('Creating backups of manager, receiver, and root frontend applications...');

  const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
  const backupTimestampDir = path.join(BACKUP_DIR, `backup-${timestamp}`);

  await ensureDir(backupTimestampDir);

  // Backup manager
  const managerBackupDir = path.join(backupTimestampDir, 'manager');
  await copyDirectory(MANAGER_DIR, managerBackupDir);
  log(`Manager backed up to: ${managerBackupDir}`, 'success');

  // Backup receiver
  const receiverBackupDir = path.join(backupTimestampDir, 'receiver');
  await copyDirectory(RECEIVER_DIR, receiverBackupDir);
  log(`Receiver backed up to: ${receiverBackupDir}`, 'success');

  // Backup existing root frontend if it exists
  const rootFrontendExists = await fileExists(ROOT_FRONTEND_DIR);
  if (rootFrontendExists) {
    const rootFrontendBackupDir = path.join(backupTimestampDir, 'root-frontend');
    await copyDirectory(ROOT_FRONTEND_DIR, rootFrontendBackupDir);
    log(`Root frontend backed up to: ${rootFrontendBackupDir}`, 'success');
  }

  // Backup backend configuration files
  const backendBackupDir = path.join(backupTimestampDir, 'backend-configs');
  await ensureDir(backendBackupDir);

  const backendFiles = ['unifiedServer.ts', 'index.ts'];
  for (const file of backendFiles) {
    const srcPath = path.join(BACKEND_DIR, file);
    const destPath = path.join(backendBackupDir, file);
    try {
      await fs.copyFile(srcPath, destPath);
      log(`Backed up backend file: ${file}`);
    } catch (error) {
      log(`Warning: Could not backup ${file}: ${error.message}`, 'warning');
    }
  }

  // Backup root package.json
  const rootPkgPath = path.join(SCRIPT_DIR, 'package.json');
  const rootPkgBackupPath = path.join(backupTimestampDir, 'root-package.json');
  try {
    await fs.copyFile(rootPkgPath, rootPkgBackupPath);
    log('Root package.json backed up');
  } catch (error) {
    log(`Warning: Could not backup root package.json: ${error.message}`, 'warning');
  }

  return backupTimestampDir;
};

const fileExists = async (filePath) => {
  try {
    await fs.access(filePath);
    return true;
  } catch {
    return false;
  }
};

// Merge package.json dependencies
const mergeDependencies = async () => {
  log('Merging package.json dependencies...');
  
  const managerPkg = await readJsonFile(path.join(MANAGER_DIR, 'package.json'));
  const receiverPkg = await readJsonFile(path.join(RECEIVER_DIR, 'package.json'));
  
  // Merge dependencies
  const mergedDependencies = {
    ...receiverPkg.dependencies,
    ...managerPkg.dependencies
  };
  
  // Merge devDependencies
  const mergedDevDependencies = {
    ...receiverPkg.devDependencies,
    ...managerPkg.devDependencies
  };
  
  // Update receiver package.json
  receiverPkg.dependencies = mergedDependencies;
  receiverPkg.devDependencies = mergedDevDependencies;
  receiverPkg.name = 'knirv-unified-controller';
  receiverPkg.description = 'Unified KNIRV Controller with integrated manager and receiver functionality';
  
  await writeJsonFile(path.join(RECEIVER_DIR, 'package.json'), receiverPkg);
  log('Dependencies merged successfully', 'success');
  
  return { managerPkg, receiverPkg };
};

// Copy manager components to receiver
const copyManagerComponents = async () => {
  log('Copying manager components to receiver...');
  
  const managerSrcDir = path.join(MANAGER_DIR, 'src');
  const receiverSrcDir = path.join(RECEIVER_DIR, 'src');
  
  // Create manager directory in receiver/src
  const managerDestDir = path.join(receiverSrcDir, 'manager');
  await ensureDir(managerDestDir);
  
  // Copy manager source files
  await copyDirectory(managerSrcDir, managerDestDir);
  log('Manager components copied successfully', 'success');
  
  // Copy manager configuration files
  const configFiles = [
    'tailwind.config.js',
    'postcss.config.js',
    'vite.config.ts',
    'tsconfig.app.json',
    'tsconfig.node.json'
  ];
  
  for (const configFile of configFiles) {
    const srcPath = path.join(MANAGER_DIR, configFile);
    const destPath = path.join(receiverSrcDir, 'manager', configFile);
    
    try {
      await fs.copyFile(srcPath, destPath);
      log(`Copied ${configFile}`);
    } catch (error) {
      log(`Warning: Could not copy ${configFile}: ${error.message}`, 'error');
    }
  }
};

// Create unified App.tsx with routing
const createUnifiedApp = async () => {
  log('Creating unified App.tsx with routing...');

  const unifiedAppContent = `import React, { useState, useEffect, useRef } from 'react';
import { BrowserRouter as Router, Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import { ComponentBridge } from './shared/ComponentBridge';

// Receiver components
import { KnirvShell } from './components/KnirvShell';
import { VoiceControl } from './components/VoiceControl';
import { NetworkStatus } from './components/NetworkStatus';
import { NRVVisualization } from './components/NRVVisualization';
import { SlidingPanel } from './components/SlidingPanel';
import { EdgeColoring } from './components/EdgeColoring';
import { AgentManager } from './components/AgentManager';
import { FabricAlgorithm } from './components/FabricAlgorithm';
import { CognitiveShellInterface } from './components/CognitiveShellInterface';
import { CognitiveState } from './sensory-shell/CognitiveEngine';

// Manager components
import UnifiedInterface from './manager/react-app/components/UnifiedInterface';
import Skills from './manager/react-app/pages/Skills';
import UDC from './manager/react-app/pages/UDC';
import WalletPage from './manager/react-app/pages/Wallet';

// Types from receiver
export interface Adaptation {
  id: string;
  type: string;
  description: string;
  timestamp: Date;
}

export interface SkillResult {
  success: boolean;
  data?: unknown;
  error?: string;
  executionTime?: number;
}

export interface NRV {
  id: string;
  problemDescription: string;
  sourceID: string;
  inputType: 'Voice' | 'Screenshot' | 'Log' | 'Camera';
  visualContext?: {
    x: number;
    y: number;
    width: number;
    height: number;
  };
  temporalContext: Date;
  severity: 'Low' | 'Medium' | 'High' | 'Critical';
  suggestedSolutionType: string;
  status: 'Identified' | 'Mapped' | 'Assigned' | 'Resolved';
}

export interface Agent {
  id: string;
  name: string;
  type: 'KNIRV-CORTEX' | 'KNIRVANA' | 'DVE';
  status: 'Available' | 'Busy' | 'Offline';
  specialization: string[];
  nrnCost: number;
}

// Navigation Button Component
const NavigationButton = ({ to, children, className = '' }) => {
  const navigate = useNavigate();
  
  return (
    <button
      onClick={() => navigate(to)}
      className={\\\`fixed top-4 right-4 z-50 bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg shadow-lg transition-all duration-200 font-medium \\\${className}\\\`}
    >
      {children}
    </button>
  );
};

// Receiver Interface Component
const ReceiverInterface = () => {
  const [shellStatus, setShellStatus] = useState<'idle' | 'processing' | 'listening' | 'error'>('idle');
  const [isVoiceActive, setIsVoiceActive] = useState(false);
  const [currentNRVs, setCurrentNRVs] = useState<NRV[]>([]);
  const [selectedNRV, setSelectedNRV] = useState<NRV | null>(null);
  const [availableAgents, setAvailableAgents] = useState<Agent[]>([]);
  const [activePanels, setActivePanels] = useState<string[]>([]);
  const [nrnBalance, setNrnBalance] = useState(1250);
  const [cognitiveMode, setCognitiveMode] = useState(false);
  const [cognitiveState, setCognitiveState] = useState<CognitiveState | null>(null);
  const [networkConnections] = useState<{
    [key: string]: 'connected' | 'disconnected' | 'connecting';
  }>({
    knirvChain: 'connected',
    knirvGraph: 'connected',
    knirvWallet: 'connected',
    knirvRouters: 'connected',
    knirvana: 'connected',
    knirvNexus: 'connected'
  });

  const shellRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    // Initialize mock agents
    const mockAgents: Agent[] = [
      {
        id: 'agent-1',
        name: 'System Diagnostics Agent',
        type: 'KNIRV-CORTEX',
        status: 'Available',
        specialization: ['error-detection', 'system-analysis'],
        nrnCost: 50
      },
      {
        id: 'agent-2', 
        name: 'UI/UX Optimization Agent',
        type: 'KNIRVANA',
        status: 'Available',
        specialization: ['interface-design', 'user-experience'],
        nrnCost: 75
      },
      {
        id: 'agent-3',
        name: 'Network Security Agent',
        type: 'DVE',
        status: 'Busy',
        specialization: ['security-analysis', 'threat-detection'],
        nrnCost: 100
      }
    ];
    setAvailableAgents(mockAgents);
  }, []);

  const handleVoiceCommand = (command: string) => {
    setShellStatus('processing');

    setTimeout(() => {
      const lowerCommand = command.toLowerCase();

      if (lowerCommand.includes('identify problems')) {
        const newNRV: NRV = {
          id: \`nrv-\${Date.now()}\`,
          problemDescription: \`User reported issue: \${command}\`,
          sourceID: 'KNIRV-CORTEX-main',
          inputType: 'Voice',
          temporalContext: new Date(),
          severity: 'Medium',
          suggestedSolutionType: 'investigation',
          status: 'Identified'
        };
        setCurrentNRVs(prev => [...prev, newNRV]);
        setShellStatus('idle');
      } else if (lowerCommand.includes('show network')) {
        setActivePanels(['network-status']);
        setShellStatus('idle');
      } else if (lowerCommand.includes('assign agents')) {
        setActivePanels(['agent-manager']);
        setShellStatus('idle');
      } else if (lowerCommand.includes('cognitive mode') || lowerCommand.includes('enable cognitive')) {
        setCognitiveMode(true);
        setActivePanels(prev => [...prev, 'cognitive-shell']);
        setShellStatus('idle');
      } else if (lowerCommand.includes('start learning')) {
        setActivePanels(prev => [...prev, 'cognitive-shell']);
        setShellStatus('idle');
      } else if (lowerCommand.includes('capture screen')) {
        handleScreenshotCapture();
        return;
      } else if (lowerCommand.includes('toggle network')) {
        handleNetworkToggle();
        setShellStatus('idle');
      } else {
        setShellStatus('idle');
      }
    }, 1500);
  };

  const handleScreenshotCapture = () => {
    setShellStatus('processing');

    setTimeout(() => {
      const newNRV: NRV = {
        id: \`nrv-\${Date.now()}\`,
        problemDescription: 'Visual anomaly detected in interface',
        sourceID: 'KNIRV-CORTEX-main',
        inputType: 'Screenshot',
        visualContext: {
          x: Math.random() * 800,
          y: Math.random() * 600,
          width: 200,
          height: 150
        },
        temporalContext: new Date(),
        severity: 'Low',
        suggestedSolutionType: 'ui-adjustment',
        status: 'Identified'
      };
      setCurrentNRVs(prev => [...prev, newNRV]);
      setShellStatus('idle');
    }, 2000);
  };

  const handleAnalyze = () => {
    setShellStatus('processing');

    setActivePanels(prev =>
      prev.includes('agent-manager')
        ? prev
        : [...prev, 'agent-manager']
    );

    setTimeout(() => {
      const newNRV: NRV = {
        id: \`nrv-\${Date.now()}\`,
        problemDescription: 'System performance degradation detected',
        sourceID: 'KNIRV-CORTEX-main',
        inputType: 'Log',
        temporalContext: new Date(),
        severity: 'Medium',
        suggestedSolutionType: 'optimization',
        status: 'Identified'
      };
      setCurrentNRVs(prev => [...prev, newNRV]);
      setShellStatus('idle');
    }, 1500);
  };

  const handleNetworkToggle = () => {
    setActivePanels(prev =>
      prev.includes('network-status')
        ? prev.filter(id => id !== 'network-status')
        : [...prev, 'network-status']
    );
  };

  const handleNRVMapping = (nrv: NRV) => {
    setCurrentNRVs(prev => prev.map(n =>
      n.id === nrv.id ? { ...n, status: 'Mapped' } : n
    ));
    setShellStatus('processing');

    setTimeout(() => {
      setShellStatus('idle');
    }, 1000);
  };

  const handleAgentAssignment = (nrv: NRV, agent: Agent) => {
    if (nrnBalance >= agent.nrnCost) {
      setNrnBalance(prev => prev - agent.nrnCost);
      setCurrentNRVs(prev => prev.map(n =>
        n.id === nrv.id ? { ...n, status: 'Assigned' } : n
      ));
      setAvailableAgents(prev => prev.map(a =>
        a.id === agent.id ? { ...a, status: 'Busy' } : a
      ));

      setTimeout(() => {
        setCurrentNRVs(prev => prev.map(n =>
          n.id === nrv.id ? { ...n, status: 'Resolved' } : n
        ));
        setAvailableAgents(prev => prev.map(a =>
          a.id === agent.id ? { ...a, status: 'Available' } : a
        ));
      }, 5000);
    }
  };

  const handleNRVClose = (nrv: NRV) => {
    setCurrentNRVs(prev => prev.filter(n => n.id !== nrv.id));
    if (selectedNRV?.id === nrv.id) {
      setSelectedNRV(null);
    }
  };

  const closePanel = (panelId: string) => {
    setActivePanels(prev => prev.filter(id => id !== panelId));
  };

  const handleCognitiveStateChange = (state: CognitiveState) => {
    setCognitiveState(state);
  };

  const handleSkillInvoked = (skillId: string, result: SkillResult) => {
    console.log('Skill invoked:', skillId, result);

    const newNRV: NRV = {
      id: \`nrv-skill-\${Date.now()}\`,
      problemDescription: \`Skill invoked: \${skillId}\`,
      sourceID: 'cognitive-shell',
      inputType: 'Voice',
      temporalContext: new Date(),
      severity: 'Low',
      suggestedSolutionType: 'skill-execution',
      status: 'Resolved'
    };
    setCurrentNRVs(prev => [...prev, newNRV]);
  };

  const handleAdaptationTriggered = (adaptation: Adaptation) => {
    console.log('Adaptation triggered:', adaptation);
    setShellStatus('processing');

    setTimeout(() => {
      setShellStatus('idle');
    }, 2000);
  };

  const getEdgeColor = () => {
    switch (shellStatus) {
      case 'processing': return '#3B82F6';
      case 'listening': return '#14B8A6';
      case 'error': return '#EF4444';
      default: return '#10B981';
    }
  };

  return (
    <div className="min-h-screen bg-gray-900 text-white relative overflow-hidden">
      <EdgeColoring color={getEdgeColor()} intensity={shellStatus !== 'idle' ? 0.8 : 0.3} />
      
      <NavigationButton to="/manager" className="bg-purple-600 hover:bg-purple-700">
        Manager Interface →
      </NavigationButton>
      
      <main ref={shellRef} className="relative w-full h-screen" role="main">
        <KnirvShell
          status={shellStatus}
          nrnBalance={nrnBalance}
          onScreenshotCapture={handleScreenshotCapture}
          onAnalyze={handleAnalyze}
          onNetworkToggle={handleNetworkToggle}
        />

        <VoiceControl
          isActive={isVoiceActive}
          onVoiceCommand={handleVoiceCommand}
          onToggle={setIsVoiceActive}
          cognitiveMode={cognitiveMode}
        />

        <NRVVisualization
          nrvs={currentNRVs}
          onNRVSelect={setSelectedNRV}
          onNRVMapping={handleNRVMapping}
          onNRVClose={handleNRVClose}
        />

        <FabricAlgorithm
          status={shellStatus}
          nrvCount={currentNRVs.length}
        />

        {/* Sliding Panels */}
        <SlidingPanel
          id="network-status"
          isOpen={activePanels.includes('network-status')}
          onClose={() => closePanel('network-status')}
          title="Network Status"
          side="right"
        >
          <NetworkStatus connections={networkConnections} />
        </SlidingPanel>

        <SlidingPanel
          id="agent-manager"
          isOpen={activePanels.includes('agent-manager')}
          onClose={() => closePanel('agent-manager')}
          title="Agent Management"
          side="left"
        >
          <AgentManager
            agents={availableAgents}
            nrvs={currentNRVs}
            selectedNRV={selectedNRV}
            onAgentAssignment={handleAgentAssignment}
            nrnBalance={nrnBalance}
          />
        </SlidingPanel>

        <SlidingPanel
          id="cognitive-shell"
          isOpen={activePanels.includes('cognitive-shell')}
          onClose={() => closePanel('cognitive-shell')}
          title="Cognitive Shell"
          side="right"
        >
          <CognitiveShellInterface
            onStateChange={handleCognitiveStateChange}
            onSkillInvoked={handleSkillInvoked}
            onAdaptationTriggered={handleAdaptationTriggered}
          />
        </SlidingPanel>

        {/* Voice Status Indicator */}
        {isVoiceActive && (
          <div className="absolute top-4 left-1/2 transform -translate-x-1/2 z-50">
            <div className="bg-teal-500 text-white px-4 py-2 rounded-full text-sm font-medium shadow-lg animate-pulse">
              <div className="flex items-center space-x-2">
                <div className="w-2 h-2 bg-white rounded-full animate-ping"></div>
                <span>Voice Active</span>
                {cognitiveMode && <span className="text-xs opacity-75">(Cognitive)</span>}
              </div>
            </div>
          </div>
        )}

        {/* Cognitive Mode Indicator */}
        {cognitiveMode && (
          <div className="absolute top-4 right-20 z-50">
            <div className="bg-purple-500 text-white px-3 py-1 rounded-full text-xs font-medium shadow-lg">
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-white rounded-full animate-pulse"></div>
                <span>Cognitive Mode</span>
              </div>
            </div>
          </div>
        )}

        {/* Status Indicator */}
        <div className="absolute bottom-4 left-4 z-40">
          <div className={\\\`px-3 py-1 rounded-full text-xs font-medium transition-all duration-300 \\\${
            shellStatus === 'idle' ? 'bg-green-500/20 text-green-400 border border-green-500/30' :
            shellStatus === 'processing' ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30' :
            shellStatus === 'listening' ? 'bg-teal-500/20 text-teal-400 border border-teal-500/30' :
            'bg-red-500/20 text-red-400 border border-red-500/30'
          }\\\`}>
            {shellStatus.charAt(0).toUpperCase() + shellStatus.slice(1)}
          </div>
        </div>
      </main>
    </div>
  );
};

// Manager Interface Wrapper
const ManagerInterface = () => {
  const [bridge, setBridge] = useState<ComponentBridge | null>(null);

  useEffect(() => {
    const componentBridge = new ComponentBridge({
      name: 'manager',
      port: 3001,
      endpoints: {
        health: '/health',
        api: '/api',
        wallet: '/wallet',
        qr: '/qr'
      },
      features: {
        qrScanning: true,
        walletIntegration: true,
        voiceControl: true,
        mobileOptimized: true
      }
    });

    setBridge(componentBridge);

    return () => {
      componentBridge.disconnect();
    };
  }, []);

  if (!bridge) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-gray-900 text-white">
        <div className="text-center">
          <div className="animate-spin rounded-full h-32 w-32 border-b-2 border-blue-500 mx-auto mb-4"></div>
          <p>Initializing KNIRV Manager...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-900 text-white relative">
      <NavigationButton to="/" className="bg-teal-600 hover:bg-teal-700">
        ← Receiver Interface
      </NavigationButton>
      
      <Routes>
        <Route path="/manager" element={<UnifiedInterface bridge={bridge} />} />
        <Route path="/manager/skills" element={<Skills />} />
        <Route path="/manager/udc" element={<UDC />} />
        <Route path="/manager/wallet" element={<WalletPage />} />
      </Routes>
    </div>
  );
};

// Main App Component
function App() {
  return (
    <Router>
      <Routes>
        <Route path="/*" element={<ReceiverInterface />} />
        <Route path="/manager/*" element={<ManagerInterface />} />
      </Routes>
    </Router>
  );
}

export default App;`;

  const appPath = path.join(RECEIVER_DIR, 'src', 'App.tsx');
  await fs.writeFile(appPath, unifiedAppContent);
  log('Unified App.tsx created successfully', 'success');
};

// Export merged receiver to root directory
const exportToRoot = async () => {
  log('Exporting merged application to root directory...');

  // Remove existing root frontend if it exists
  const rootFrontendExists = await fileExists(ROOT_FRONTEND_DIR);
  if (rootFrontendExists) {
    await fs.rm(ROOT_FRONTEND_DIR, { recursive: true, force: true });
    log('Removed existing root frontend directory');
  }

  // Copy merged receiver to root as 'frontend'
  await copyDirectory(RECEIVER_DIR, ROOT_FRONTEND_DIR);
  log('Merged application exported to root/frontend', 'success');

  // Update the root frontend package.json
  const rootFrontendPkgPath = path.join(ROOT_FRONTEND_DIR, 'package.json');
  const rootFrontendPkg = await readJsonFile(rootFrontendPkgPath);

  // Update package.json for root deployment
  rootFrontendPkg.name = 'knirv-controller-frontend';
  rootFrontendPkg.description = 'KNIRV Controller Unified Frontend (Manager + Receiver)';

  await writeJsonFile(rootFrontendPkgPath, rootFrontendPkg);
  log('Root frontend package.json updated', 'success');
};

// Update backend configuration for root frontend
const updateBackendConfig = async () => {
  log('Updating backend configuration for root frontend...');

  const unifiedServerPath = path.join(BACKEND_DIR, 'unifiedServer.ts');
  let serverContent = await fs.readFile(unifiedServerPath, 'utf8');

  // Update the receiver dist path to point to root frontend
  serverContent = serverContent.replace(
    /this\.receiverDistPath = path\.join\(rootDir, 'receiver', 'dist'\);/,
    "this.receiverDistPath = path.join(rootDir, 'frontend', 'dist');"
  );

  // Update log messages
  serverContent = serverContent.replace(
    /📱 Receiver frontend available at:/,
    '📱 Unified frontend available at:'
  );

  serverContent = serverContent.replace(
    /GET  \/ - Receiver Frontend/,
    'GET  / - Unified Frontend (Manager + Receiver)'
  );

  serverContent = serverContent.replace(
    /Frontend not built\. Run "npm run build:receiver" first\./,
    'Frontend not built. Run "npm run build:frontend" first.'
  );

  await fs.writeFile(unifiedServerPath, serverContent);
  log('Backend unified server updated', 'success');
};

// Update root package.json scripts
const updateRootPackageScripts = async () => {
  log('Updating root package.json scripts...');

  const rootPkgPath = path.join(SCRIPT_DIR, 'package.json');
  const rootPkg = await readJsonFile(rootPkgPath);

  // Add new scripts for unified frontend
  const newScripts = {
    ...rootPkg.scripts,
    'dev:frontend': 'cd frontend && npm run dev',
    'build:frontend': 'cd frontend && npm run build',
    'test:frontend': 'cd frontend && npm test',
    'lint:frontend': 'cd frontend && npm run lint',
    'install:frontend': 'cd frontend && npm install',

    // Update existing scripts
    'dev': 'npm run build:frontend && npm run build:backend && nodemon --exec \'node dist/unifiedServer.js\'',
    'dev:unified': 'npm run build:frontend && npm run build:backend && nodemon --exec \'node dist/unifiedServer.js\'',
    'build:unified': 'npm run build:backend && npm run build:frontend',
    'build:all': 'npm run build:backend && npm run build:frontend',
    'start': 'node dist/unifiedServer.js',

    // Update test scripts
    'test:all': 'npm run test:unit && npm run test:integration && npm run test:e2e && npm run test:frontend',
    'lint:all': 'npm run lint:backend && npm run lint:tests && npm run lint:frontend',
    'install:all': 'npm run install:frontend'
  };

  rootPkg.scripts = newScripts;

  // Update description
  rootPkg.description = 'Unified KNIRV Controller with integrated frontend (manager + receiver), backend, and agent-core compiler';

  await writeJsonFile(rootPkgPath, rootPkg);
  log('Root package.json scripts updated', 'success');
};

// Install dependencies
const installDependencies = async () => {
  log('Installing dependencies for unified frontend...');

  try {
    execSync('npm install', {
      cwd: ROOT_FRONTEND_DIR,
      stdio: 'inherit'
    });
    log('Frontend dependencies installed successfully', 'success');

    // Also install root dependencies
    execSync('npm install', {
      cwd: SCRIPT_DIR,
      stdio: 'inherit'
    });
    log('Root dependencies installed successfully', 'success');
  } catch (error) {
    log(`Error installing dependencies: ${error.message}`, 'error');
    throw error;
  }
};

// Main execution function
const main = async () => {
  try {
    log('Starting KNIRV Controller Manager-Receiver merge and root export process...');

    // Step 1: Create backups
    const backupDir = await createBackups();

    // Step 2: Merge dependencies
    await mergeDependencies();

    // Step 3: Copy manager components
    await copyManagerComponents();

    // Step 4: Create unified App.tsx
    await createUnifiedApp();

    // Step 5: Export merged receiver to root directory
    await exportToRoot();

    // Step 6: Update backend configuration
    await updateBackendConfig();

    // Step 7: Update root package.json scripts
    await updateRootPackageScripts();

    // Step 8: Install dependencies
    await installDependencies();

    log('✅ Merge and export completed successfully!', 'success');
    log(`📁 Backups stored in: ${backupDir}`);
    log('🏠 Unified application exported to: ./frontend/');
    log('🔧 Backend updated to serve from root frontend');
    log('📦 Root package.json scripts updated');
    log('');
    log('🚀 Available commands:');
    log('  npm run dev              - Start unified development server');
    log('  npm run build:frontend   - Build frontend only');
    log('  npm run build:unified    - Build both backend and frontend');
    log('  npm start                - Start production server');
    log('');
    log('🌐 Access points:');
    log('  http://localhost:3000/           - Receiver Interface');
    log('  http://localhost:3000/manager    - Manager Interface');
    log('  http://localhost:3000/health     - Health check');
    log('  http://localhost:3000/api        - Backend API');

  } catch (error) {
    log(`❌ Merge failed: ${error.message}`, 'error');
    process.exit(1);
  }
};

// Run the script
if (import.meta.url === `file://${process.argv[1]}`) {
  main();
}

export {
  main,
  createBackups,
  mergeDependencies,
  copyManagerComponents,
  createUnifiedApp,
  exportToRoot,
  updateBackendConfig,
  updateRootPackageScripts
};
