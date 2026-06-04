'use client';

import React from 'react';
import Link from 'next/link';
import { Button } from '@/components/ui/button';
import { 
  Shield, 
  Cpu, 
  Database, 
  Activity, 
  Lock, 
  Network,
  Terminal,
  AlertTriangle,
  ChevronRight,
  Zap,
  Layers,
  Scan,
  FileSearch
} from 'lucide-react';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

interface HeroProps {
  onGetStarted?: () => void;
}

const Hero = ({ onGetStarted }: HeroProps) => {
  return (
    <TooltipProvider delayDuration={100}>
      <div className="relative overflow-hidden bg-[#0a0a0c] min-h-screen flex items-center">
        {/* Background Effects */}
        <div className="fixed inset-0 overflow-hidden pointer-events-none opacity-20">
          <div className="absolute top-[-10%] left-[-10%] w-[40%] h-[40%] bg-blue-600/20 blur-[120px] rounded-full" />
          <div className="absolute bottom-[-10%] right-[-10%] w-[40%] h-[40%] bg-indigo-600/10 blur-[120px] rounded-full" />
          <div className="absolute inset-0 bg-grid-white/[0.02] bg-[length:30px_30px]" />
        </div>

        {/* Content */}
        <div className="relative z-10 max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-24 md:py-32">
          <div className="text-center space-y-8">
            {/* Badge */}
            <div className="inline-flex items-center space-x-2 px-4 py-2 rounded-full border border-blue-500/30 bg-blue-500/10">
              <Zap className="w-4 h-4 text-blue-400" />
              <span className="text-xs font-bold text-blue-400 uppercase tracking-widest">KNIRV Nexus v2.0</span>
            </div>

            {/* Main Title */}
            <h1 className="text-5xl md:text-6xl lg:text-7xl font-extrabold text-white tracking-tight leading-tight">
              Build Your{' '}
              <span className="text-transparent bg-clip-text bg-gradient-to-r from-blue-400 to-cyan-400">
                Sovereign Agent
              </span>
              <br />
              <span className="text-transparent bg-clip-text bg-gradient-to-r from-cyan-400 to-blue-500">
                Black Box
              </span>
            </h1>

            {/* Declaration Statement */}
            <div className="max-w-4xl mx-auto">
              <p className="text-xl md:text-2xl text-slate-300 leading-relaxed">
                Unlike standard gateways that simply route API calls, the{' '}
                <span className="text-blue-400 font-semibold">KNIRV Nexus</span> provides a{' '}
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span className="border-b-2 border-dotted border-blue-500/50 cursor-help hover:border-blue-400 transition-colors">
                      Persistent State Store
                    </span>
                  </TooltipTrigger>
                  <TooltipContent side="bottom" className="max-w-md p-4 bg-slate-900 border-slate-700">
                    <div className="space-y-3">
                      <div className="flex items-center space-x-2">
                        <Database className="w-5 h-5 text-blue-400" />
                        <span className="font-bold text-white">Persistent State Store</span>
                      </div>
                      <p className="text-sm text-slate-300">
                        A Persistent State Store and a Secure Execution Environment for your data fabric.
                      </p>
                      <ul className="space-y-1">
                        <li className="text-xs text-slate-400 flex items-start">
                          <ChevronRight className="w-3 h-3 mr-1 mt-0.5 text-blue-500 flex-shrink-0" />
                          Immediate &quot;Chain of Thought&quot; and active session state
                        </li>
                        <li className="text-xs text-slate-400 flex items-start">
                          <ChevronRight className="w-3 h-3 mr-1 mt-0.5 text-blue-500 flex-shrink-0" />
                          High-speed retrieval of recent logs and system metrics
                        </li>
                        <li className="text-xs text-slate-400 flex items-start">
                          <ChevronRight className="w-3 h-3 mr-1 mt-0.5 text-blue-500 flex-shrink-0" />
                          Long-term factual storage and forensic audit trails
                        </li>
                      </ul>
                    </div>
                  </TooltipContent>
                </Tooltip>
                {' '}and a{' '}
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span className="border-b-2 border-dotted border-green-500/50 cursor-help hover:border-green-400 transition-colors">
                      Secure Execution Environment
                    </span>
                  </TooltipTrigger>
                  <TooltipContent side="bottom" className="max-w-md p-4 bg-slate-900 border-slate-700">
                    <div className="space-y-3">
                      <div className="flex items-center space-x-2">
                        <Shield className="w-5 h-5 text-green-400" />
                        <span className="font-bold text-white">Kernel-Level Security Probes</span>
                      </div>
                      <p className="text-sm text-slate-300">Real-time kernel-level monitoring and intervention</p>
                      <ul className="space-y-1">
                        <li className="text-xs text-slate-400 flex items-start">
                          <ChevronRight className="w-3 h-3 mr-1 mt-0.5 text-green-500 flex-shrink-0" />
                          Captures plain-text LLM prompts before they are encrypted for the provider
                        </li>
                        <li className="text-xs text-slate-400 flex items-start">
                          <ChevronRight className="w-3 h-3 mr-1 mt-0.5 text-green-500 flex-shrink-0" />
                          Monitors system calls to ensure agents calling a &quot;Search Tool&quot; aren&apos;t attempting to launch a reverse shell
                        </li>
                      </ul>
                    </div>
                  </TooltipContent>
                </Tooltip>
              </p>
            </div>

            {/* Action Buttons - Moved to middle under sub-heading */}
            <div className="flex flex-col sm:flex-row gap-4 justify-center py-4">
              <Tooltip>
                <TooltipTrigger asChild>
                  <div>
                    {onGetStarted ? (
                      <Button
                        onClick={onGetStarted}
                        size="lg"
                        className="bg-blue-600 hover:bg-blue-500 text-white font-bold px-8 py-6 text-lg h-auto"
                      >
                        <Zap className="w-5 h-5 mr-2" />
                        Initialize Data Fabric
                      </Button>
                    ) : (
                      <Button asChild size="lg" className="bg-blue-600 hover:bg-blue-500 text-white font-bold px-8 py-6 text-lg h-auto">
                        <Link href="/dashboard">
                          <Zap className="w-5 h-5 mr-2" />
                          Initialize Data Fabric
                        </Link>
                      </Button>
                    )}
                  </div>
                </TooltipTrigger>
                <TooltipContent side="bottom" className="max-w-xs p-4 bg-slate-900 border-slate-700">
                  <div className="space-y-2">
                    <p className="text-sm text-white font-bold">Start Your Sovereign Journey</p>
                    <p className="text-xs text-slate-400">Create your secure Data Fabric with persistent state storage and kernel-protected execution environment.</p>
                  </div>
                </TooltipContent>
              </Tooltip>

              <Tooltip>
                <TooltipTrigger asChild>
                  <div>
                    <Button asChild size="lg" variant="outline" className="border-white/20 text-slate-300 hover:bg-white/10 hover:text-white px-8 py-6 text-lg h-auto">
                      <Link href="/documentation">
                        <Terminal className="w-5 h-5 mr-2" />
                        Explore Documentation
                      </Link>
                    </Button>
                  </div>
                </TooltipTrigger>
                <TooltipContent side="bottom" className="max-w-xs p-4 bg-slate-900 border-slate-700">
                  <div className="space-y-2">
                    <p className="text-sm text-white font-bold">Learn the Architecture</p>
                    <p className="text-xs text-slate-400">Deep dive into the KNIRV Nexus, kernel probes, memory-efficient data streaming, and the Fabric data flow.</p>
                  </div>
                </TooltipContent>
              </Tooltip>
            </div>

            {/* Feature Grid - Hover Cards */}
            <div className="grid md:grid-cols-3 gap-6 max-w-5xl mx-auto pt-8">
              {/* Persistent State */}
              <Tooltip>
                <TooltipTrigger asChild>
                  <div className="group p-6 bg-white/5 border border-white/10 rounded-2xl hover:border-blue-500/50 hover:bg-white/10 transition-all cursor-pointer">
                    <div className="flex items-center justify-center w-12 h-12 bg-blue-500/10 rounded-xl mb-4 group-hover:bg-blue-500/20 transition-colors">
                      <Database className="w-6 h-6 text-blue-400" />
                    </div>
                    <h3 className="font-bold text-white mb-2">Persistent State</h3>
                    <p className="text-sm text-slate-400">Chain of Thought, session state, and forensic audit trails</p>
                  </div>
                </TooltipTrigger>
                <TooltipContent side="bottom" className="max-w-sm p-4 bg-slate-900 border-slate-700">
                  <div className="space-y-3">
                    <div className="flex items-center space-x-2">
                      <Database className="w-5 h-5 text-blue-400" />
                      <span className="font-bold text-white">Persistent State Store</span>
                    </div>
                    <ul className="space-y-2">
                      <li className="text-xs text-slate-300 flex items-start">
                        <Activity className="w-3 h-3 mr-2 mt-0.5 text-blue-500" />
                        Immediate Chain of Thought and active session state
                      </li>
                      <li className="text-xs text-slate-300 flex items-start">
                        <Activity className="w-3 h-3 mr-2 mt-0.5 text-blue-500" />
                        High-speed retrieval of recent logs and system metrics
                      </li>
                      <li className="text-xs text-slate-300 flex items-start">
                        <Activity className="w-3 h-3 mr-2 mt-0.5 text-blue-500" />
                        Long-term factual storage and forensic audit trails
                      </li>
                    </ul>
                  </div>
                </TooltipContent>
              </Tooltip>

              {/* Security Probes */}
              <Tooltip>
                <TooltipTrigger asChild>
                  <div className="group p-6 bg-white/5 border border-white/10 rounded-2xl hover:border-green-500/50 hover:bg-white/10 transition-all cursor-pointer">
                    <div className="flex items-center justify-center w-12 h-12 bg-green-500/10 rounded-xl mb-4 group-hover:bg-green-500/20 transition-colors">
                      <Shield className="w-6 h-6 text-green-400" />
                    </div>
                    <h3 className="font-bold text-white mb-2">Kernel-Level Security</h3>
                    <p className="text-sm text-slate-400">Deep system monitoring at the OS level</p>
                  </div>
                </TooltipTrigger>
                <TooltipContent side="bottom" className="max-w-sm p-4 bg-slate-900 border-slate-700">
                  <div className="space-y-3">
                    <div className="flex items-center space-x-2">
                      <Shield className="w-5 h-5 text-green-400" />
                      <span className="font-bold text-white">Kernel-Level Security Monitoring</span>
                    </div>
                    <div className="space-y-2">
                      <div className="p-2 bg-green-500/5 rounded border border-green-500/20">
                        <div className="text-xs font-bold text-green-400 mb-1">Prompt Inspection</div>
                        <p className="text-xs text-slate-400">Captures plain-text LLM prompts before they are encrypted for the provider</p>
                      </div>
                      <div className="p-2 bg-green-500/5 rounded border border-green-500/20">
                        <div className="text-xs font-bold text-green-400 mb-1">System Call Monitoring</div>
                        <p className="text-xs text-slate-400">Monitors network and execution calls to ensure agents calling a &quot;Search Tool&quot; aren&apos;t attempting to launch a reverse shell</p>
                      </div>
                    </div>
                  </div>
                </TooltipContent>
              </Tooltip>

              {/* Kill Switch */}
              <Tooltip>
                <TooltipTrigger asChild>
                  <div className="group p-6 bg-white/5 border border-white/10 rounded-2xl hover:border-red-500/50 hover:bg-white/10 transition-all cursor-pointer">
                    <div className="flex items-center justify-center w-12 h-12 bg-red-500/10 rounded-xl mb-4 group-hover:bg-red-500/20 transition-colors">
                      <AlertTriangle className="w-6 h-6 text-red-400" />
                    </div>
                    <h3 className="font-bold text-white mb-2">Security Kill Switch</h3>
                    <p className="text-sm text-slate-400">Intent-Action correlation with drift detection</p>
                  </div>
                </TooltipTrigger>
                <TooltipContent side="bottom" className="max-w-sm p-4 bg-slate-900 border-slate-700">
                  <div className="space-y-3">
                    <div className="flex items-center space-x-2">
                      <AlertTriangle className="w-5 h-5 text-red-400" />
                      <span className="font-bold text-white">Security Kill Switch</span>
                    </div>
                    <p className="text-xs text-slate-300">Intent-Action Correlation with Drift Detection</p>
                    <ul className="space-y-2">
                      <li className="text-xs text-slate-400 flex items-start">
                        <Lock className="w-3 h-3 mr-2 mt-0.5 text-red-500" />
                        Maps agent predicted tool call against actual system calls captured by the kernel
                      </li>
                      <li className="text-xs text-slate-400 flex items-start">
                        <Lock className="w-3 h-3 mr-2 mt-0.5 text-red-500" />
                        Blocks calls and revokes sessions if Agent Action deviates from Intent
                      </li>
                    </ul>
                  </div>
                </TooltipContent>
              </Tooltip>

              {/* Zero-Copy Memory */}
              <Tooltip>
                <TooltipTrigger asChild>
                  <div className="group p-6 bg-white/5 border border-white/10 rounded-2xl hover:border-blue-500/50 hover:bg-white/10 transition-all cursor-pointer">
                    <div className="flex items-center justify-center w-12 h-12 bg-blue-500/10 rounded-xl mb-4 group-hover:bg-blue-500/20 transition-colors">
                      <Cpu className="w-6 h-6 text-blue-400" />
                    </div>
                    <h3 className="font-bold text-white mb-2">Zero-Copy Memory</h3>
                    <p className="text-sm text-slate-400">Direct binary context streaming to agent memory</p>
                  </div>
                </TooltipTrigger>
                <TooltipContent side="bottom" className="max-w-sm p-4 bg-slate-900 border-slate-700">
                  <div className="space-y-3">
                    <div className="flex items-center space-x-2">
                      <Cpu className="w-5 h-5 text-blue-400" />
                      <span className="font-bold text-white">Zero-Copy Memory Architecture</span>
                    </div>
                    <p className="text-xs text-slate-300">Direct memory access without serialization overhead</p>
                    <div className="space-y-2">
                      <div className="p-2 bg-blue-500/5 rounded border border-blue-500/20">
                        <div className="text-xs font-bold text-blue-400 mb-1">Schema Discovery</div>
                        <p className="text-xs text-slate-400">Returns location and schema of specific memory batches for efficient access</p>
                      </div>
                      <div className="p-2 bg-blue-500/5 rounded border border-blue-500/20">
                        <div className="text-xs font-bold text-blue-400 mb-1">Direct Memory Streaming</div>
                        <p className="text-xs text-slate-400">Streams binary context directly to the agent&apos;s memory space without copying</p>
                      </div>
                    </div>
                  </div>
                </TooltipContent>
              </Tooltip>

              {/* The Fabric */}
              <Tooltip>
                <TooltipTrigger asChild>
                  <div className="group p-6 bg-white/5 border border-white/10 rounded-2xl hover:border-amber-500/50 hover:bg-white/10 transition-all cursor-pointer md:col-span-2">
                    <div className="flex items-center justify-center w-12 h-12 bg-amber-500/10 rounded-xl mb-4 group-hover:bg-amber-500/20 transition-colors">
                      <Network className="w-6 h-6 text-amber-400" />
                    </div>
                    <h3 className="font-bold text-white mb-2">Living Data Flow</h3>
                    <p className="text-sm text-slate-400">Ingestion → Processing → Context Injection → Direct Handover → Audit</p>
                  </div>
                </TooltipTrigger>
                <TooltipContent side="bottom" className="max-w-lg p-4 bg-slate-900 border-slate-700">
                  <div className="space-y-3">
                    <div className="flex items-center space-x-2">
                      <Layers className="w-5 h-5 text-amber-400" />
                      <span className="font-bold text-white">Living Data Flow</span>
                    </div>
                    <div className="grid grid-cols-2 gap-2">
                      <div className="p-2 bg-amber-500/5 rounded border border-amber-500/20">
                        <div className="text-xs font-bold text-amber-400 mb-1">1. Ingestion</div>
                        <p className="text-xs text-slate-400">Agent performs task, Nexus captures via protocol interceptor</p>
                      </div>
                      <div className="p-2 bg-amber-500/5 rounded border border-amber-500/20">
                        <div className="text-xs font-bold text-amber-400 mb-1">2. Processing</div>
                        <p className="text-xs text-slate-400">In-memory compute kernels filter data based on relevance scores</p>
                      </div>
                      <div className="p-2 bg-amber-500/5 rounded border border-amber-500/20">
                        <div className="text-xs font-bold text-amber-400 mb-1">3. Context Injection</div>
                        <p className="text-xs text-slate-400">Filtered Memory mapped into structured data tables</p>
                      </div>
                      <div className="p-2 bg-amber-500/5 rounded border border-amber-500/20">
                        <div className="text-xs font-bold text-amber-400 mb-1">4. Direct Handover</div>
                        <p className="text-xs text-slate-400">Memory pointer provided directly to Agent (Python/TS) with no serialization</p>
                      </div>
                    </div>
                    <div className="p-2 bg-amber-500/5 rounded border border-amber-500/20">
                      <div className="text-xs font-bold text-amber-400 mb-1">5. Audit</div>
                      <p className="text-xs text-slate-400">Finalized state written to optimized columnar storage format using concurrent processing</p>
                    </div>
                  </div>
                </TooltipContent>
              </Tooltip>
            </div>
          </div>
        </div>

        {/* Footer Meta */}
        <footer className="absolute bottom-0 left-0 right-0 p-6 text-center">
          <div className="inline-flex items-center space-x-4 px-6 py-2 rounded-full border border-white/5 bg-white/5 text-[10px] mono text-slate-500 font-bold uppercase tracking-[0.2em]">
            <span className="flex h-2 w-2 rounded-full bg-green-500 animate-pulse" />
            <span>Nexus Network Secure</span>
            <span className="h-3 w-[1px] bg-white/10" />
            <span>Sovereign Encryption Active</span>
          </div>
        </footer>
      </div>
    </TooltipProvider>
  );
};

export default Hero;
