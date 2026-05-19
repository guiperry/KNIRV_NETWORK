'use client';

import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/lib/auth-context';

export default function ArenaPage() {
  const router = useRouter();
  const { user, isLoading } = useAuth();
  const [arenaReady, setArenaReady] = useState(false);
  const [loadingError, setLoadingError] = useState<string | null>(null);

  // Verify auth and check arena availability
  useEffect(() => {
    if (isLoading) return;

    if (!user) {
      // Redirect to login if not authenticated
      router.push('/login');
      return;
    }

    // Check if arena service is available by probing /arena endpoint
    const checkArenaHealth = async () => {
      try {
        const response = await fetch('/arena/health', {
          method: 'GET',
          headers: {
            'Accept': 'application/json',
          },
        });

        if (response.ok || response.status === 404) {
          // Arena service is running (either /arena/health exists or 404 means service is up)
          setArenaReady(true);
        } else {
          setLoadingError('Arena service is not responding correctly');
          setArenaReady(false);
        }
      } catch (error) {
        console.error('Failed to reach arena service:', error);
        setLoadingError('Failed to connect to arena service. Please ensure it is running.');
        setArenaReady(false);
      }
    };

    checkArenaHealth();
  }, [user, isLoading, router]);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-[#030a18]">
        <div className="text-center">
          <div className="animate-spin mb-4">
            <div className="w-16 h-16 border-4 border-indigo-500/30 border-t-indigo-500 rounded-full"></div>
          </div>
          <p className="text-indigo-400 font-mono text-sm">Loading authentication...</p>
        </div>
      </div>
    );
  }

  if (!user) {
    return null; // Will redirect via useEffect
  }

  if (loadingError) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-[#030a18]">
        <div className="max-w-md w-full mx-4">
          <div className="border border-red-500/30 bg-red-500/5 rounded-lg p-6">
            <h2 className="text-red-400 font-bold mb-2">Arena Unavailable</h2>
            <p className="text-red-300/80 text-sm mb-4">{loadingError}</p>
            <button
              onClick={() => router.push('/')}
              className="w-full px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white rounded transition-colors text-sm font-mono"
            >
              Return to Dashboard
            </button>
          </div>
        </div>
      </div>
    );
  }

  if (!arenaReady) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-[#030a18]">
        <div className="text-center">
          <div className="animate-spin mb-4">
            <div className="w-16 h-16 border-4 border-indigo-500/30 border-t-indigo-500 rounded-full"></div>
          </div>
          <p className="text-indigo-400 font-mono text-sm">Initializing KNIRVARENA...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="w-full h-screen bg-[#030a18] overflow-hidden">
      {/* Arena is served via iframe from /arena/* endpoints */}
      <iframe
        src="/arena/"
        className="w-full h-full border-none"
        title="KNIRVARENA - 3D Game Arena"
        sandbox="allow-same-origin allow-scripts allow-popups allow-forms allow-pointer-lock"
        style={{
          backgroundColor: '#030a18',
        }}
      />

      {/* Fallback UI for when iframe fails to load */}
      <noscript>
        <div className="absolute inset-0 flex items-center justify-center bg-[#030a18]">
          <div className="text-center">
            <p className="text-red-400 font-mono mb-4">JavaScript is required to run KNIRVARENA</p>
          </div>
        </div>
      </noscript>
    </div>
  );
}
