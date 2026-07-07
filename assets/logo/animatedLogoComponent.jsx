import React, { useState, useEffect } from 'react';

const KnirvNetworkLogo = () => {
  const [glowIntensity, setGlowIntensity] = useState(0.5);
  
  useEffect(() => {
    const interval = setInterval(() => {
      setGlowIntensity(prev => 0.3 + Math.sin(Date.now() * 0.002) * 0.2);
    }, 50);
    return () => clearInterval(interval);
  }, []);

  return (
    <div className="flex items-center justify-center w-full h-full bg-gradient-to-br from-gray-900 via-black to-purple-900 p-8">
      <div className="flex items-center space-x-6">
        {/* Neural Network Visualization */}
        <div className="relative">
          <svg 
            width="120" 
            height="120" 
            viewBox="0 0 120 120" 
            className="drop-shadow-2xl"
            style={{
              filter: `drop-shadow(0 0 ${20 * glowIntensity}px #00f5ff) drop-shadow(0 0 ${40 * glowIntensity}px #8b5cf6)`
            }}
          >
            {/* Central neuron soma */}
            <circle 
              cx="60" 
              cy="60" 
              r="8" 
              fill="url(#somaGradient)" 
              stroke="#00f5ff" 
              strokeWidth="1.5"
            />
            <circle 
              cx="60" 
              cy="60" 
              r="12" 
              fill="none" 
              stroke="url(#glowRing)" 
              strokeWidth="1" 
              opacity={glowIntensity * 0.6}
              className="animate-pulse"
            />
            
            {/* Main branching dendrites - natural organic curves */}
            {/* Upper left branch */}
            <path 
              d="M55 52 C50 45, 45 40, 40 35 C38 32, 35 30, 32 28 C30 26, 27 25, 25 22" 
              stroke="url(#dendriteGradient)" 
              strokeWidth="2" 
              fill="none" 
              opacity="0.9"
            />
            <path 
              d="M40 35 C38 32, 35 31, 33 28" 
              stroke="url(#dendriteGradient)" 
              strokeWidth="1.5" 
              fill="none" 
              opacity="0.8"
            />
            <path 
              d="M35 30 C32 28, 30 25, 28 23" 
              stroke="url(#dendriteGradient)" 
              strokeWidth="1" 
              fill="none" 
              opacity="0.7"
            />
            
            {/* Upper right branch */}
            <path 
              d="M65 52 C70 45, 75 40, 80 35 C82 32, 85 30, 88 28 C90 26, 93 25, 95 22" 
              stroke="url(#dendriteGradient)" 
              strokeWidth="2" 
              fill="none" 
              opacity="0.9"
            />
            <path 
              d="M80 35 C82 32, 85 31, 87 28" 
              stroke="url(#dendriteGradient)" 
              strokeWidth="1.5" 
              fill="none" 
              opacity="0.8"
            />
            <path 
              d="M85 30 C88 28, 90 25, 92 23" 
              stroke="url(#dendriteGradient)" 
              strokeWidth="1" 
              fill="none" 
              opacity="0.7"
            />
            
            {/* Lower left branch */}
            <path 
              d="M55 68 C50 75, 45 80, 40 85 C38 88, 35 90, 32 92 C30 94, 27 95, 25 98" 
              stroke="url(#dendriteGradient)" 
              strokeWidth="2" 
              fill="none" 
              opacity="0.9"
            />
            <path 
              d="M40 85 C38 88, 35 89, 33 92" 
              stroke="url(#dendriteGradient)" 
              strokeWidth="1.5" 
              fill="none" 
              opacity="0.8"
            />
            <path 
              d="M35 90 C32 92, 30 95, 28 97" 
              stroke="url(#dendriteGradient)" 
              strokeWidth="1" 
              fill="none" 
              opacity="0.7"
            />
            
            {/* Lower right branch */}
            <path 
              d="M65 68 C70 75, 75 80, 80 85 C82 88, 85 90, 88 92 C90 94, 93 95, 95 98" 
              stroke="url(#dendriteGradient)" 
              strokeWidth="2" 
              fill="none" 
              opacity="0.9"
            />
            <path 
              d="M80 85 C82 88, 85 89, 87 92" 
              stroke="url(#dendriteGradient)" 
              strokeWidth="1.5" 
              fill="none" 
              opacity="0.8"
            />
            <path 
              d="M85 90 C88 92, 90 95, 92 97" 
              stroke="url(#dendriteGradient)" 
              strokeWidth="1" 
              fill="none" 
              opacity="0.7"
            />
            
            {/* Left side horizontal branch */}
            <path 
              d="M52 60 C45 58, 38 56, 30 55 C27 54, 24 53, 20 52" 
              stroke="url(#dendriteGradient)" 
              strokeWidth="2" 
              fill="none" 
              opacity="0.9"
            />
            <path 
              d="M30 55 C27 53, 24 52, 22 50" 
              stroke="url(#dendriteGradient)" 
              strokeWidth="1.5" 
              fill="none" 
              opacity="0.8"
            />
            
            {/* Right side horizontal branch */}
            <path 
              d="M68 60 C75 58, 82 56, 90 55 C93 54, 96 53, 100 52" 
              stroke="url(#dendriteGradient)" 
              strokeWidth="2" 
              fill="none" 
              opacity="0.9"
            />
            <path 
              d="M90 55 C93 53, 96 52, 98 50" 
              stroke="url(#dendriteGradient)" 
              strokeWidth="1.5" 
              fill="none" 
              opacity="0.8"
            />
            
            {/* Synaptic terminals with glowing effect */}
            <g style={{ opacity: glowIntensity }}>
              <circle cx="25" cy="22" r="2" fill="#00f5ff" />
              <circle cx="95" cy="22" r="2" fill="#00f5ff" />
              <circle cx="25" cy="98" r="2" fill="#00f5ff" />
              <circle cx="95" cy="98" r="2" fill="#00f5ff" />
              <circle cx="20" cy="52" r="2" fill="#00f5ff" />
              <circle cx="100" cy="52" r="2" fill="#00f5ff" />
              
              {/* Smaller synaptic terminals */}
              <circle cx="33" cy="28" r="1.5" fill="#8b5cf6" />
              <circle cx="87" cy="28" r="1.5" fill="#8b5cf6" />
              <circle cx="33" cy="92" r="1.5" fill="#8b5cf6" />
              <circle cx="87" cy="92" r="1.5" fill="#8b5cf6" />
              <circle cx="22" cy="50" r="1.5" fill="#8b5cf6" />
              <circle cx="98" cy="50" r="1.5" fill="#8b5cf6" />
            </g>
            
            {/* Electrical activity - moving along dendrites */}
            <g style={{ opacity: glowIntensity }}>
              <circle cx={60 - (glowIntensity * 30)} cy={60 - (glowIntensity * 20)} r="1.5" fill="#fff" />
              <circle cx={60 + (glowIntensity * 30)} cy={60 - (glowIntensity * 20)} r="1.5" fill="#fff" />
              <circle cx={60 - (glowIntensity * 30)} cy={60 + (glowIntensity * 20)} r="1.5" fill="#fff" />
              <circle cx={60 + (glowIntensity * 30)} cy={60 + (glowIntensity * 20)} r="1.5" fill="#fff" />
              <circle cx={60 - (glowIntensity * 25)} cy="60" r="1.5" fill="#fff" />
              <circle cx={60 + (glowIntensity * 25)} cy="60" r="1.5" fill="#fff" />
            </g>
            
            {/* Central electrical activity */}
            <circle 
              cx="60" 
              cy="60" 
              r="4" 
              fill="#fff" 
              opacity={glowIntensity * 0.8}
              className="animate-pulse"
            />
            
            <defs>
              <radialGradient id="somaGradient" cx="30%" cy="30%">
                <stop offset="0%" stopColor="#fff" />
                <stop offset="30%" stopColor="#00f5ff" />
                <stop offset="70%" stopColor="#0080ff" />
                <stop offset="100%" stopColor="#1a1a2e" />
              </radialGradient>
              <linearGradient id="dendriteGradient" x1="0%" y1="0%" x2="100%" y2="100%">
                <stop offset="0%" stopColor="#00f5ff" />
                <stop offset="30%" stopColor="#0080ff" />
                <stop offset="70%" stopColor="#8b5cf6" />
                <stop offset="100%" stopColor="#1a1a2e" />
              </linearGradient>
              <radialGradient id="glowRing" cx="50%" cy="50%">
                <stop offset="0%" stopColor="transparent" />
                <stop offset="80%" stopColor="transparent" />
                <stop offset="100%" stopColor="#00f5ff" />
              </radialGradient>
            </defs>
          </svg>
        </div>
        
        {/* Company Name */}
        <div className="flex flex-col">
          {/* KNIRV */}
          <div className="relative">
            <h1 
              className="text-6xl font-bold tracking-wider text-transparent bg-clip-text bg-gradient-to-r from-purple-400 via-pink-400 to-purple-600"
              style={{
                fontFamily: "'Orbitron', 'Arial Black', sans-serif",
                textShadow: `0 0 ${10 * glowIntensity}px rgba(168, 85, 247, 0.5)`
              }}
            >
              KNIRV
            </h1>
            {/* Subtle underline effect */}
            <div 
              className="h-1 bg-gradient-to-r from-transparent via-purple-500 to-transparent mt-1"
              style={{ opacity: glowIntensity }}
            ></div>
          </div>
          
          {/* Network */}
          <div className="relative mt-2">
            <h2 
              className="text-2xl font-light tracking-[0.3em] text-gray-300 uppercase"
              style={{
                fontFamily: "'Rajdhani', 'Arial', sans-serif",
                textShadow: '0 2px 4px rgba(0,0,0,0.5)'
              }}
            >
              Network
            </h2>
            {/* Connection dots */}
            <div className="flex space-x-2 mt-1">
              <div className="w-2 h-2 bg-purple-400 rounded-full animate-pulse"></div>
              <div className="w-2 h-2 bg-pink-400 rounded-full animate-pulse" style={{ animationDelay: '0.2s' }}></div>
              <div className="w-2 h-2 bg-purple-400 rounded-full animate-pulse" style={{ animationDelay: '0.4s' }}></div>
              <div className="w-2 h-2 bg-pink-400 rounded-full animate-pulse" style={{ animationDelay: '0.6s' }}></div>
            </div>
          </div>
        </div>
      </div>
      
      {/* Background tech pattern */}
      <div className="absolute inset-0 opacity-10 pointer-events-none">
        <svg width="100%" height="100%">
          <defs>
            <pattern id="grid" width="50" height="50" patternUnits="userSpaceOnUse">
              <path d="M 50 0 L 0 0 0 50" fill="none" stroke="#8b5cf6" strokeWidth="1"/>
            </pattern>
          </defs>
          <rect width="100%" height="100%" fill="url(#grid)" />
        </svg>
      </div>
    </div>
  );
};

export default KnirvNetworkLogo;