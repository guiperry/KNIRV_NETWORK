'use client';

import React, { Suspense, useEffect, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import ConsolePanel from '@/components/dashboard/console-panel';
import { Box, Terminal as LucideTerminal } from 'lucide-react';

function StandaloneTerminal() {
  const searchParams = useSearchParams();
  const nodeId = searchParams.get('nodeId') || undefined;
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
    // Dark background for the body in pop-out mode
    document.body.style.backgroundColor = '#03050a';
    return () => {
      document.body.style.backgroundColor = '';
    };
  }, []);

  if (!mounted) return null;

  return (
    <div className="fixed inset-0 bg-[#03050a] flex flex-col overflow-hidden">
      {/* Pop-out specific header for branding */}
      <div className="h-12 border-b border-blue-600/30 bg-slate-900/80 backdrop-blur-sm p-4 flex items-center justify-between z-50">
        <div className="flex items-center space-x-2">
          <Box className="w-5 h-5 text-blue-500 animate-pulse" />
          <h1 className="text-[10px] font-black tracking-tighter text-blue-100 uppercase">
            KNIRV Sovereign Terminal
          </h1>
          <div className="h-4 w-px bg-slate-800 mx-2" />
          <span className="text-[9px] font-mono text-slate-500 uppercase">
            Context: {nodeId || 'GLOBAL FABRIC'}
          </span>
        </div>
        <div className="flex items-center space-x-2">
           <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse mr-1" />
           <span className="text-[9px] font-black text-green-400">SESSION: ACTIVE</span>
        </div>
      </div>

      {/* The Terminal Panel - Force it to fill the screen */}
      <div className="flex-1 relative">
        <ConsolePanel 
          isOpen={true} 
          onClose={() => window.close()} 
          nodeId={nodeId}
          isMonitorOpen={false}
          isStandalone={true}
        />
      </div>

      {/* Simple Footer */}
      <div className="h-6 bg-slate-950 border-t border-blue-600/10 flex items-center px-4 justify-between">
         <span className="text-[8px] text-slate-600 font-mono">SECURE TEE TUNNEL // AES-256-GCM</span>
         <span className="text-[8px] text-slate-600 font-mono">v1.1.0-STABLE</span>
      </div>
    </div>
  );
}

export default function TerminalPage() {
  return (
    <Suspense fallback={
      <div className="h-screen w-screen bg-[#03050a] flex items-center justify-center">
        <div className="text-center space-y-4">
          <LucideTerminal className="w-12 h-12 text-blue-500 animate-pulse mx-auto" />
          <p className="text-[10px] font-black text-blue-400 uppercase tracking-widest">
            Initializing Secure Context...
          </p>
        </div>
      </div>
    }>
      <StandaloneTerminal />
    </Suspense>
  );
}
