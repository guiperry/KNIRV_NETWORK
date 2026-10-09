import React from 'react';
import { useState, useEffect } from 'react';
import { AlertTriangle, Activity, Zap, Target, Clock, TrendingUp, CheckCircle, AlertCircle, ExternalLink } from 'lucide-react';
import { NRV } from '../../App';
import { getKNIRVSERVERClient, type ErrorNodeTestSuite } from '../../services/KNIRVSERVERClient';
import { knirvEngineBrowserLink, openInKnirvEngine } from '../../services/knirvEngineLink';

interface ErrorNodeModalProps {
  isOpen: boolean;
  onClose: () => void;
  nrvs: NRV[];
  selectedNRV: NRV | null;
  nrnBalance: number;
  onDeployAgent: (nrv: NRV) => void;
}

interface ErrorNode {
  id: string;
  title: string;
  description: string;
  severity: 'low' | 'medium' | 'high' | 'critical';
  status: 'active' | 'solving' | 'resolved';
  complexity: number;
  estimatedTime: number; // in minutes
  nrnReward: number;
  nrnCost: number;
  requiredSkills: string[];
  progress?: number;
  lastActivity?: string;
  /** Network KNIRVGRAPH id; tests attach only to this, never the local id. */
  networkErrorNodeId?: string;
}

export const ErrorNodeModal: React.FC<ErrorNodeModalProps> = ({
  isOpen,
  onClose,
  nrvs,
  selectedNRV: _selectedNRV,
  nrnBalance: _nrnBalance,
  onDeployAgent
}) => {
  const [errorNodes, setErrorNodes] = useState<ErrorNode[]>([]);
  const [selectedErrorNode, setSelectedErrorNode] = useState<ErrorNode | null>(null);
  const [testSuite, setTestSuite] = useState<ErrorNodeTestSuite | null>(null);
  const [isLoadingTests, setIsLoadingTests] = useState(false);
  const [isSavingTest, setIsSavingTest] = useState(false);
  const [testError, setTestError] = useState<string | null>(null);
  const [testInput, setTestInput] = useState('');
  const [expectedOutput, setExpectedOutput] = useState('');
  const [testDescription, setTestDescription] = useState('');
  const [engineState, setEngineState] = useState<'idle' | 'opening' | 'opened' | 'unhandled'>('idle');

  // Generate mock error nodes from NRVs
  useEffect(() => {
    const generateErrorNodes = (): ErrorNode[] => {
      return nrvs.map((nrv, index) => ({
        id: nrv.id,
        networkErrorNodeId: nrv.networkErrorNodeId,
        title: `Error ${index + 1}: ${nrv.problemDescription.split(' ').slice(0, 3).join(' ')}...`,
        description: nrv.problemDescription,
        severity: nrv.severity as 'low' | 'medium' | 'high' | 'critical',
        status: Math.random() > 0.7 ? 'solving' : 'active',
        complexity: Math.floor(Math.random() * 3) + 1,
        estimatedTime: Math.floor(Math.random() * 45) + 15, // 15-60 minutes
        nrnReward: Math.floor(Math.random() * 100) + 50, // 50-150 NRN
        nrnCost: Math.floor(Math.random() * 75) + 25, // 25-100 NRN
        requiredSkills: ['Debugging', nrv.suggestedSolutionType],
        progress: Math.random() > 0.7 ? Math.floor(Math.random() * 100) : undefined,
        lastActivity: new Date(Date.now() - Math.random() * 86400000).toLocaleTimeString()
      }));
    };

    if (isOpen) {
      setErrorNodes(generateErrorNodes());
    }
  }, [nrvs, isOpen]);

  const selectErrorNode = async (node: ErrorNode) => {
    setSelectedErrorNode(node);
    setEngineState('idle');
    setTestSuite(null);
    setTestError(null);
    if (!node.networkErrorNodeId) return;
    setIsLoadingTests(true);
    try {
      setTestSuite(await getKNIRVSERVERClient().getErrorNodeTests(node.networkErrorNodeId));
    } catch (error) {
      setTestError(error instanceof Error ? error.message : 'Could not load this error node\'s tests.');
    } finally {
      setIsLoadingTests(false);
    }
  };

  const contributeTest = async () => {
    if (!selectedErrorNode?.networkErrorNodeId || !testInput.trim() || !expectedOutput.trim()) return;
    setIsSavingTest(true);
    setTestError(null);
    try {
      const suite = await getKNIRVSERVERClient().contributeErrorNodeTest(selectedErrorNode.networkErrorNodeId, {
        id: `arena-${Date.now()}`,
        input: testInput.trim(),
        expected: expectedOutput.trim(),
        description: testDescription.trim() || undefined,
      });
      setTestSuite(suite);
      setTestInput('');
      setExpectedOutput('');
      setTestDescription('');
    } catch (error) {
      setTestError(error instanceof Error ? error.message : 'Could not contribute the test.');
    } finally {
      setIsSavingTest(false);
    }
  };

  // Hands the node to KNIRVENGINE's error-node analysis: the desktop engine
  // via knirvengine://, or a browser-launched engine when none answers.
  const openInEngine = async (node: ErrorNode) => {
    if (!node.networkErrorNodeId) return;
    setEngineState('opening');
    setEngineState((await openInKnirvEngine(node.networkErrorNodeId)) ? 'opened' : 'unhandled');
  };

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical': return 'from-rose-500 to-red-600';
      case 'high': return 'from-orange-500 to-red-600';
      case 'medium': return 'from-amber-400 to-orange-600';
      case 'low': return 'from-yellow-300 to-amber-500';
      default: return 'from-gray-500 to-slate-600';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'solving': return Activity;
      case 'resolved': return CheckCircle;
      default: return AlertTriangle;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'solving': return 'text-blue-400';
      case 'resolved': return 'text-emerald-400';
      default: return 'text-amber-400';
    }
  };

  const handleDeploySolution = (errorNode: ErrorNode) => {
    // Find corresponding NRV and deploy solution
    const nrv = nrvs.find(n => n.id === errorNode.id);
    if (nrv) {
      onDeployAgent(nrv);
    }
  };

  const activeErrors = errorNodes.filter(en => en.status === 'active');
  const solvingErrors = errorNodes.filter(en => en.status === 'solving');
  const resolvedErrors = errorNodes.filter(en => en.status === 'resolved');
  const totalRewards = errorNodes.reduce((sum, en) => sum + (en.status === 'resolved' ? en.nrnReward : 0), 0);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4">
      <div className="w-full max-w-6xl max-h-[90vh] bg-slate-950 rounded-2xl border border-slate-800 shadow-2xl overflow-hidden">
        {/* Header */}
        <div className="sticky top-0 z-30 backdrop-blur supports-[backdrop-filter]:bg-slate-950/70 border-b border-slate-800">
          <div className="flex items-center justify-between px-6 py-4">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-2xl bg-gradient-to-br from-rose-500 to-orange-600 shadow-lg shadow-rose-800/30">
                <AlertTriangle className="w-6 h-6" />
              </div>
              <div>
                <h1 className="text-xl md:text-2xl font-semibold tracking-tight">Error Node Management</h1>
                <p className="text-xs text-slate-400">Monitor and resolve network errors</p>
              </div>
            </div>
            <button
              onClick={onClose}
              className="p-2 rounded-lg hover:bg-slate-800 transition-colors"
            >
              <span className="text-slate-400 text-xl">×</span>
            </button>
          </div>
        </div>

        <div className="p-6">
          {/* Stats Overview */}
          <div className="grid grid-cols-4 gap-4 mb-6">
            <div className="p-4 rounded-xl border border-slate-800 bg-slate-900/60">
              <div className="flex items-center gap-2 mb-2">
                <AlertTriangle className="w-4 h-4 text-amber-400" />
                <span className="text-xs text-slate-400">Active</span>
              </div>
              <div className="text-xl font-bold text-slate-200">{activeErrors.length}</div>
            </div>
            
            <div className="p-4 rounded-xl border border-slate-800 bg-slate-900/60">
              <div className="flex items-center gap-2 mb-2">
                <Activity className="w-4 h-4 text-blue-400" />
                <span className="text-xs text-slate-400">Solving</span>
              </div>
              <div className="text-xl font-bold text-blue-400">{solvingErrors.length}</div>
            </div>
            
            <div className="p-4 rounded-xl border border-slate-800 bg-slate-900/60">
              <div className="flex items-center gap-2 mb-2">
                <CheckCircle className="w-4 h-4 text-emerald-400" />
                <span className="text-xs text-slate-400">Resolved</span>
              </div>
              <div className="text-xl font-bold text-emerald-400">{resolvedErrors.length}</div>
            </div>
            
            <div className="p-4 rounded-xl border border-slate-800 bg-slate-900/60">
              <div className="flex items-center gap-2 mb-2">
                <Zap className="w-4 h-4 text-yellow-400" />
                <span className="text-xs text-slate-400">Total Rewards</span>
              </div>
              <div className="text-xl font-bold text-yellow-400">{totalRewards} NRN</div>
            </div>
          </div>

          <div className="grid grid-cols-12 gap-6">
            {/* Error Nodes List */}
            <div className="col-span-12 lg:col-span-8">
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <h2 className="text-lg font-medium text-slate-200">Error Nodes</h2>
                  <div className="flex items-center gap-2 text-sm text-slate-400">
                    <span>{errorNodes.length} total</span>
                  </div>
                </div>
                
                <div className="space-y-3 max-h-96 overflow-y-auto">
                  {errorNodes.map((errorNode) => {
                    const StatusIcon = getStatusIcon(errorNode.status);
                    return (
                      <div
                        key={errorNode.id}
                        className="p-4 rounded-xl border border-slate-800 bg-slate-900/60 hover:border-slate-700 transition-all cursor-pointer"
                        onClick={() => void selectErrorNode(errorNode)}
                      >
                        <div className="flex items-start justify-between mb-2">
                          <div className="flex-1">
                            <div className="flex items-center gap-2 mb-1">
                              <StatusIcon className={`w-4 h-4 ${getStatusColor(errorNode.status)}`} />
                              <span className="text-sm font-medium text-slate-200">{errorNode.title}</span>
                            </div>
                            <p className="text-xs text-slate-400 line-clamp-2">{errorNode.description}</p>
                          </div>
                          <div className="flex items-center gap-2">
                            <span className={`text-xs px-2 py-1 rounded-full border bg-gradient-to-br ${getSeverityColor(errorNode.severity)}`}>
                              {errorNode.severity}
                            </span>
                          </div>
                        </div>

                        <div className="flex items-center justify-between mt-3">
                          <div className="flex items-center gap-4 text-xs text-slate-400">
                            <div className="flex items-center gap-1">
                              <Target className="w-3 h-3" />
                              <span>Complexity {errorNode.complexity}</span>
                            </div>
                            <div className="flex items-center gap-1">
                              <Clock className="w-3 h-3" />
                              <span>{errorNode.estimatedTime}m</span>
                            </div>
                            {errorNode.progress !== undefined && (
                              <div className="flex items-center gap-1">
                                <TrendingUp className="w-3 h-3 text-blue-400" />
                                <span className="text-blue-400">{errorNode.progress}%</span>
                              </div>
                            )}
                          </div>
                          <div className="flex items-center gap-2">
                            <span className="text-sm font-medium text-yellow-400">{errorNode.nrnReward} NRN</span>
                            {errorNode.status === 'active' && (
                              <button
                                onClick={(e) => {
                                  e.stopPropagation();
                                  handleDeploySolution(errorNode);
                                }}
                                className="px-3 py-1 rounded-lg bg-indigo-600 text-white text-xs hover:bg-indigo-700"
                              >
                                Deploy Solution
                              </button>
                            )}
                          </div>
                        </div>

                        {errorNode.progress !== undefined && (
                          <div className="mt-2">
                            <div className="w-full bg-slate-700 rounded-full h-1">
                              <div 
                                className="bg-blue-500 h-1 rounded-full transition-all duration-500"
                                style={{ width: `${errorNode.progress}%` }}
                              />
                            </div>
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              </div>
            </div>

            {/* Selected Error Details */}
            <div className="col-span-12 lg:col-span-4">
              {selectedErrorNode ? (
                <div className="space-y-4">
                  <h3 className="text-lg font-medium text-slate-200">Error Details</h3>
                  
                  <div className="p-4 rounded-xl border border-slate-800 bg-slate-900/60">
                    <div className="flex items-center gap-2 mb-3">
                      {(() => {
                        const StatusIcon = getStatusIcon(selectedErrorNode.status);
                        return <StatusIcon className={`w-5 h-5 ${getStatusColor(selectedErrorNode.status)}`} />;
                      })()}
                      <span className="font-medium text-slate-200">{selectedErrorNode.title}</span>
                    </div>
                    
                    <div className="space-y-3">
                      <div>
                        <div className="text-xs text-slate-400 mb-1">Description</div>
                        <p className="text-sm text-slate-300">{selectedErrorNode.description}</p>
                      </div>

                      <div className="space-y-2">
                        <button
                          type="button"
                          onClick={() => void openInEngine(selectedErrorNode)}
                          disabled={!selectedErrorNode.networkErrorNodeId || engineState === 'opening'}
                          title={selectedErrorNode.networkErrorNodeId ? 'Analyze this error node, its files and tests in KNIRVENGINE' : 'Only errors registered on the network KNIRVGRAPH can be opened in KNIRVENGINE'}
                          className="w-full inline-flex items-center justify-center gap-2 rounded-md border border-rose-500/50 bg-rose-500/10 px-2 py-1.5 text-xs font-medium text-rose-100 hover:bg-rose-500/20 disabled:cursor-not-allowed disabled:opacity-50"
                        >
                          <ExternalLink className="w-3.5 h-3.5" />
                          {engineState === 'opening' ? 'Opening KNIRVENGINE…' : 'Open in KNIRVENGINE'}
                        </button>
                        {engineState === 'opened' && (
                          <p className="text-xs text-emerald-300">Opened in KNIRVENGINE&apos;s error-node analysis.</p>
                        )}
                        {engineState === 'unhandled' && selectedErrorNode.networkErrorNodeId && (
                          <p className="text-xs text-amber-300">
                            The KNIRVENGINE desktop app didn&apos;t respond.{' '}
                            <a
                              href={knirvEngineBrowserLink(selectedErrorNode.networkErrorNodeId)}
                              target="_blank"
                              rel="noopener noreferrer"
                              className="underline text-cyan-300 hover:text-cyan-200"
                            >
                              Open it in the browser engine instead
                            </a>
                          </p>
                        )}
                      </div>

                      <div className="rounded-lg border border-slate-800 bg-slate-950/50 p-3 space-y-3">
                        <div className="flex items-center justify-between gap-2">
                          <div>
                            <div className="text-xs font-medium text-slate-200">Shared error-node tests</div>
                            <p className="text-xs text-slate-500">The first 8 tests seal the same suite used for swarm grading and badge exams.</p>
                          </div>
                          <span className="shrink-0 text-xs font-medium text-cyan-300">
                            {isLoadingTests ? 'Loading…' : `${testSuite?.tests.length ?? 0}/${testSuite?.required ?? 8}`}
                          </span>
                        </div>
                        {!selectedErrorNode.networkErrorNodeId ? (
                          <p className="text-xs text-amber-300">
                            This error only exists in this client, so tests can&apos;t attach to it yet. Errors submitted
                            while connected to KNIRVSERVER are registered on the network KNIRVGRAPH and take tests there.
                          </p>
                        ) : testSuite?.status === 'sealed' ? (
                          <p className="text-xs text-emerald-300">Suite sealed. Its inputs are now the shared evaluation bar.</p>
                        ) : (
                          <>
                            <input
                              aria-label="Test description"
                              value={testDescription}
                              onChange={(event) => setTestDescription(event.target.value)}
                              placeholder="What does this test cover? (optional)"
                              className="w-full rounded-md border border-slate-700 bg-slate-900 px-2 py-1.5 text-xs text-slate-100 placeholder:text-slate-600"
                            />
                            <textarea
                              aria-label="Test input"
                              value={testInput}
                              onChange={(event) => setTestInput(event.target.value)}
                              placeholder="Input sent to a candidate solution"
                              rows={2}
                              className="w-full rounded-md border border-slate-700 bg-slate-900 px-2 py-1.5 text-xs text-slate-100 placeholder:text-slate-600"
                            />
                            <textarea
                              aria-label="Expected output"
                              value={expectedOutput}
                              onChange={(event) => setExpectedOutput(event.target.value)}
                              placeholder="Expected output (kept private by KNIRVGRAPH)"
                              rows={2}
                              className="w-full rounded-md border border-slate-700 bg-slate-900 px-2 py-1.5 text-xs text-slate-100 placeholder:text-slate-600"
                            />
                            <button
                              type="button"
                              onClick={() => void contributeTest()}
                              disabled={isSavingTest || !testInput.trim() || !expectedOutput.trim()}
                              className="w-full rounded-md bg-cyan-700 px-2 py-1.5 text-xs font-medium text-white disabled:cursor-not-allowed disabled:opacity-50 hover:bg-cyan-600"
                            >
                              {isSavingTest ? 'Saving test…' : 'Add test to this error node'}
                            </button>
                          </>
                        )}
                        {testError && <p role="alert" className="text-xs text-rose-300">{testError}</p>}
                      </div>
                      
                      <div>
                        <div className="text-xs text-slate-400 mb-2">Required Skills</div>
                        <div className="flex flex-wrap gap-2">
                          {selectedErrorNode.requiredSkills.map((skill, idx) => (
                            <span key={idx} className="text-xs px-2 py-1 rounded-full bg-slate-800 text-slate-300">
                              {skill}
                            </span>
                          ))}
                        </div>
                      </div>
                      
                      <div className="grid grid-cols-2 gap-3">
                        <div>
                          <div className="text-xs text-slate-400">Severity</div>
                          <div className={`text-sm font-medium capitalize ${
                            selectedErrorNode.severity === 'critical' ? 'text-red-400' :
                            selectedErrorNode.severity === 'high' ? 'text-orange-400' :
                            selectedErrorNode.severity === 'medium' ? 'text-amber-400' :
                            'text-yellow-400'
                          }`}>
                            {selectedErrorNode.severity}
                          </div>
                        </div>
                        <div>
                          <div className="text-xs text-slate-400">Complexity</div>
                          <div className="text-sm font-medium text-slate-300">{selectedErrorNode.complexity}</div>
                        </div>
                        <div>
                          <div className="text-xs text-slate-400">Est. Time</div>
                          <div className="text-sm font-medium text-slate-300">{selectedErrorNode.estimatedTime}m</div>
                        </div>
                        <div>
                          <div className="text-xs text-slate-400">Reward</div>
                          <div className="text-sm font-medium text-yellow-400">{selectedErrorNode.nrnReward} NRN</div>
                        </div>
                      </div>
                      
                      {selectedErrorNode.lastActivity && (
                        <div>
                          <div className="text-xs text-slate-400">Last Activity</div>
                          <div className="text-sm text-slate-300">{selectedErrorNode.lastActivity}</div>
                        </div>
                      )}
                      
                      {selectedErrorNode.status === 'active' && (
                        <button
                          onClick={() => handleDeploySolution(selectedErrorNode)}
                          className="w-full py-2 px-4 rounded-lg bg-indigo-600 text-white font-medium hover:bg-indigo-700 transition-colors"
                        >
                          Deploy Solution ({selectedErrorNode.nrnCost} NRN)
                        </button>
                      )}
                    </div>
                  </div>
                </div>
              ) : (
                <div className="text-center py-8">
                  <AlertCircle className="w-8 h-8 mx-auto mb-2 opacity-50 text-slate-600" />
                  <p className="text-sm text-slate-500">Select an error node to view details</p>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
