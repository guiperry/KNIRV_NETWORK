import React, { useState, useEffect, useRef } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Star, Zap, Cpu, Network, Shield, Wallet, Router, Server, Database, Globe, X, ChevronRight } from 'lucide-react';

interface MenuItem {
  id: string;
  name: string;
  icon: React.ReactNode;
  description: string;
  details: string[];
  angle: number;
  radius: number;
  ring: 'inner' | 'outer';
  color: string;
}

const App: React.FC = () => {
  const [hoveredItem, setHoveredItem] = useState<string | null>(null);
  const [selectedItem, setSelectedItem] = useState<string | null>(null);
  const [animationPhase, setAnimationPhase] = useState<'stars' | 'supernova' | 'sirius' | 'menu'>('stars');
  const [mousePos, setMousePos] = useState({ x: 0, y: 0 });

  const menuItems: MenuItem[] = [
    { 
      id: 'pay', 
      name: 'KNIRV PAY', 
      icon: <Wallet size={32} />, 
      description: 'Payment Gateway',
      details: ['Cross-chain payments', 'Multi-asset support', 'Instant settlements', 'Low fees'],
      angle: 0, 
      radius: 180, 
      ring: 'inner', 
      color: '#ffffff' 
    },
    { 
      id: 'chain', 
      name: 'KNIRV CHAIN', 
      icon: <Database size={32} />, 
      description: 'Blockchain Layer',
      details: ['Skill registry', 'Node transformation', 'Mining capabilities', 'Consensus mechanism'],
      angle: 180, 
      radius: 180, 
      ring: 'inner', 
      color: '#ffffff' 
    },
    { 
      id: 'nexus', 
      name: 'KNIRV NEXUS', 
      icon: <Network size={32} />, 
      description: 'Distributed Validation',
      details: ['DVE environment', 'Sovereign layers', 'IBC communication', 'Collective intelligence'],
      angle: 270, 
      radius: 180, 
      ring: 'inner', 
      color: '#ffffff' 
    },
    { 
      id: 'oracle', 
      name: 'KNIRV ORACLE', 
      icon: <Server size={32} />, 
      description: 'Cross-Chain Hub',
      details: ['Governance authority', 'Token economics', 'IBC transfers', 'Network parameters'],
      angle: 90, 
      radius: 180, 
      ring: 'inner', 
      color: '#ffffff' 
    },
    { 
      id: 'gateway', 
      name: 'KNIRV GATEWAY', 
      icon: <Globe size={32} />, 
      description: 'API Gateway',
      details: ['Documentation site', 'RESTful APIs', 'Developer portal', 'Rate limiting'],
      angle: 45, 
      radius: 280, 
      ring: 'outer', 
      color: '#00f0ff' 
    },
    { 
      id: 'wallet', 
      name: 'KNIRV WALLET', 
      icon: <Wallet size={32} />, 
      description: 'Non-Custodial Wallet',
      details: ['XION Meta Accounts', 'Browser extension', 'Mobile app', 'Hardware wallet support'],
      angle: 225, 
      radius: 280, 
      ring: 'outer', 
      color: '#00f0ff' 
    },
    { 
      id: 'router', 
      name: 'KNIRV ROUTER', 
      icon: <Router size={32} />, 
      description: 'P2P Network',
      details: ['Proof-of-Connectivity', 'Node discovery', 'Message routing', 'Network backbone'],
      angle: 315, 
      radius: 280, 
      ring: 'outer', 
      color: '#00f0ff' 
    },
    { 
      id: 'controller', 
      name: 'KNIRV CONTROLLER', 
      icon: <Cpu size={32} />, 
      description: 'AI Gateway',
      details: ['Autonomous AI', 'WASM runtime', 'User gateway', 'Skill execution'],
      angle: 135, 
      radius: 280, 
      ring: 'outer', 
      color: '#00f0ff' 
    }
  ];

  const getItemPosition = (item: MenuItem) => {
    const radian = ((item.angle - 90) * Math.PI) / 180;
    const x = window.innerWidth / 2 + Math.cos(radian) * item.radius;
    const y = window.innerHeight / 2 + Math.sin(radian) * item.radius;
    return { x, y };
  };

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      setMousePos({ x: e.clientX, y: e.clientY });
    };
    window.addEventListener('mousemove', handleMouseMove);
    return () => window.removeEventListener('mousemove', handleMouseMove);
  }, []);

  useEffect(() => {
    const timer1 = setTimeout(() => setAnimationPhase('supernova'), 2000);
    const timer2 = setTimeout(() => setAnimationPhase('sirius'), 3000);
    const timer3 = setTimeout(() => setAnimationPhase('menu'), 5000);
    
    return () => {
      clearTimeout(timer1);
      clearTimeout(timer2);
      clearTimeout(timer3);
    };
  }, []);

  const selectedItemData = menuItems.find(item => item.id === selectedItem);

  return (
    <div className="relative w-full h-full bg-gradient-to-b from-blue-950 to-black overflow-hidden">
      {/* Stars Background */}
      <AnimatePresence>
        {animationPhase !== 'menu' && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: animationPhase === 'stars' ? 1 : 0.3 }}
            exit={{ opacity: 0 }}
            className="absolute inset-0 pointer-events-none"
          >
            {Array.from({ length: 200 }).map((_, i) => (
              <motion.div
                key={i}
                className="absolute"
                style={{
                  left: `${Math.random() * 100}%`,
                  top: `${Math.random() * 100}%`,
                }}
                animate={{
                  opacity: [0.2, 1, 0.2],
                  scale: [1, 1.2, 1],
                }}
                transition={{
                  duration: 2 + Math.random() * 3,
                  repeat: Infinity,
                  delay: Math.random() * 2,
                }}
              >
                <Star 
                  size={Math.random() * 4 + 1} 
                  color="#a0d8ef" 
                  fill="#a0d8ef"
                />
              </motion.div>
            ))}
          </motion.div>
        )}
      </AnimatePresence>

      {/* Supernova */}
      <AnimatePresence>
        {animationPhase === 'supernova' && (
          <motion.div
            className="absolute top-1/2 left-1/2 w-80 h-80 -translate-x-1/2 -translate-y-1/2"
            style={{
              background: 'radial-gradient(circle, #ffffff, #00aaff, transparent)',
              borderRadius: '50%',
              boxShadow: '0 0 100px 50px #00aaff',
            }}
            animate={{ scale: [0.1, 3], opacity: [1, 0] }}
            transition={{ duration: 1 }}
            exit={{ opacity: 0 }}
          />
        )}
      </AnimatePresence>

      {/* Sirius Star */}
      <AnimatePresence>
        {(animationPhase === 'sirius' || animationPhase === 'menu') && (
          <motion.div
            className="absolute top-1/2 left-1/2 w-40 h-40 -translate-x-1/2 -translate-y-1/2"
            style={{
              borderRadius: '50%',
              border: '5px solid #00f0ff',
              background: 'radial-gradient(circle, rgba(0, 240, 255, 0.3), rgba(0, 20, 60, 1))',
              boxShadow: '0 0 50px 20px #00f0ff, inset 0 0 30px 10px #00f0ff',
              display: 'flex',
              justifyContent: 'center',
              alignItems: 'center',
            }}
            animate={{ rotate: [0, 360] }}
            transition={{ duration: 4, repeat: Infinity, ease: 'linear' }}
            exit={{ scale: 0, opacity: 0 }}
          >
            <div className="absolute">
              <Zap color="#00f0ff" size={40} className="absolute -top-5 left-1/2 -translate-x-1/2" />
              <Zap color="#00f0ff" size={40} className="absolute -bottom-5 left-1/2 -translate-x-1/2 rotate-180" />
              <Zap color="#00f0ff" size={30} className="absolute -left-4 top-1/2 -translate-y-1/2 rotate-90" />
              <Zap color="#00f0ff" size={30} className="absolute -right-4 top-1/2 -translate-y-1/2 -rotate-90" />
            </div>
          </motion.div>
        )}
      </AnimatePresence>

      {/* Interactive Menu */}
      <AnimatePresence>
        {animationPhase === 'menu' && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="absolute inset-0"
          >
            {/* Title */}
            <motion.h1
              className="absolute top-20 left-1/2 -translate-x-1/2 text-white text-5xl font-mono tracking-wider z-20"
              style={{ textShadow: '0 0 30px #00f0ff' }}
              initial={{ opacity: 0, y: -50 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.5, duration: 0.8 }}
            >
              KNIRVANA
            </motion.h1>

            {/* Menu Items */}
            {menuItems.map((item, index) => {
              const position = getItemPosition(item);
              const isHovered = hoveredItem === item.id;
              
              return (
                <motion.div
                  key={item.id}
                  className="absolute w-32 h-32 -translate-x-1/2 -translate-y-1/2 cursor-pointer z-10"
                  style={{
                    left: position.x,
                    top: position.y,
                  }}
                  initial={{ scale: 0, opacity: 0 }}
                  animate={{ 
                    scale: isHovered ? 1.2 : 1, 
                    opacity: 1,
                  }}
                  transition={{ 
                    delay: 0.8 + index * 0.1,
                    type: 'spring',
                    stiffness: 300,
                  }}
                  whileHover={{ scale: 1.2, zIndex: 100 }}
                  onHoverStart={() => setHoveredItem(item.id)}
                  onHoverEnd={() => setHoveredItem(null)}
                  onClick={() => setSelectedItem(item.id)}
                >
                  <div
                    className="w-full h-full rounded-full border-2 backdrop-blur-md flex flex-col items-center justify-center relative overflow-hidden"
                    style={{
                      borderColor: item.color,
                      background: isHovered 
                        ? `radial-gradient(circle, ${item.color}22, ${item.color}11)`
                        : `radial-gradient(circle, ${item.color}11, transparent)`,
                      boxShadow: isHovered 
                        ? `0 0 40px 20px ${item.color}88, inset 0 0 20px ${item.color}44`
                        : `0 0 20px 10px ${item.color}44, inset 0 0 10px ${item.color}22`,
                    }}
                  >
                    {/* Animated Ring */}
                    {isHovered && (
                      <motion.div
                        className="absolute inset-0 rounded-full border-2"
                        style={{ borderColor: item.color }}
                        animate={{ scale: [1, 1.2, 1], opacity: [0.8, 0.4, 0.8] }}
                        transition={{ duration: 1, repeat: Infinity }}
                      />
                    )}
                    
                    {/* Icon */}
                    <div className={`${item.color} mb-2 z-10`} style={{ color: item.color }}>
                      {item.icon}
                    </div>
                    
                    {/* Name */}
                    <div
                      className="text-xs font-bold text-center z-10"
                      style={{ 
                        color: item.color,
                        textShadow: `0 0 10px ${item.color}`,
                      }}
                    >
                      {item.name.split(' ')[1]}
                    </div>
                  </div>
                  
                  {/* Hover Description */}
                  {isHovered && (
                    <motion.div
                      className="absolute -top-12 left-1/2 -translate-x-1/2 bg-black/90 px-4 py-2 rounded-full text-xs whitespace-nowrap border z-50"
                      style={{
                        borderColor: item.color,
                        color: item.color,
                        boxShadow: `0 4px 20px ${item.color}66`,
                      }}
                      initial={{ opacity: 0, y: 10 }}
                      animate={{ opacity: 1, y: 0 }}
                    >
                      {item.description}
                    </motion.div>
                  )}
                </motion.div>
              );
            })}

            {/* Constellation Lines */}
            <svg className="absolute inset-0 w-full h-full pointer-events-none" style={{ zIndex: 1 }}>
              {/* Orbital Rings */}
              <circle
                cx="50%"
                cy="50%"
                r={180}
                fill="none"
                stroke="#00f0ff"
                strokeWidth="2"
                strokeDasharray="10,5"
                opacity={0.6}
              />
              <circle
                cx="50%"
                cy="50%"
                r={280}
                fill="none"
                stroke="#00f0ff"
                strokeWidth="2"
                strokeDasharray="10,5"
                opacity={0.4}
              />
              
              {/* Connection Lines */}
              {menuItems.slice(0, 4).map((item, index) => {
                const position = getItemPosition(item);
                const nextItem = menuItems[(index + 1) % 4];
                const nextPosition = getItemPosition(nextItem);
                return (
                  <line
                    key={`inner-${item.id}`}
                    x1={position.x}
                    y1={position.y}
                    x2={nextPosition.x}
                    y2={nextPosition.y}
                    stroke="#00f0ff"
                    strokeWidth="1"
                    opacity="0.3"
                    strokeDasharray="3,3"
                  />
                );
              })}
              
              {menuItems.slice(4).map((item, index) => {
                const position = getItemPosition(item);
                const nextItem = menuItems[4 + ((index + 1) % 4)];
                const nextPosition = getItemPosition(nextItem);
                return (
                  <line
                    key={`outer-${item.id}`}
                    x1={position.x}
                    y1={position.y}
                    x2={nextPosition.x}
                    y2={nextPosition.y}
                    stroke="#00f0ff"
                    strokeWidth="1"
                    opacity="0.2"
                    strokeDasharray="3,3"
                  />
                );
              })}
            </svg>
          </motion.div>
        )}
      </AnimatePresence>

      {/* Selected Item Modal */}
      <AnimatePresence>
        {selectedItem && selectedItemData && (
          <motion.div
            className="fixed inset-0 bg-black/80 backdrop-blur-sm flex items-center justify-center z-50"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            onClick={() => setSelectedItem(null)}
          >
            <motion.div
              className="bg-gradient-to-b from-blue-900/50 to-black/80 border-2 border-cyan-400 rounded-3xl p-8 max-w-md mx-4"
              style={{
                boxShadow: '0 0 50px #00f0ff',
                backdropFilter: 'blur(20px)',
              }}
              initial={{ scale: 0.8, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              exit={{ scale: 0.8, opacity: 0 }}
              onClick={(e) => e.stopPropagation()}
            >
              <div className="flex justify-between items-start mb-6">
                <div>
                  <h2 className="text-3xl font-bold text-white mb-2">{selectedItemData.name}</h2>
                  <p className="text-cyan-400">{selectedItemData.description}</p>
                </div>
                <button
                  className="text-gray-400 hover:text-white transition-colors"
                  onClick={() => setSelectedItem(null)}
                >
                  <X size={24} />
                </button>
              </div>
              
              <div className="mb-6">
                <h3 className="text-cyan-400 font-semibold mb-3 flex items-center">
                  <ChevronRight size={16} className="mr-1" />
                  Key Features
                </h3>
                <ul className="space-y-2">
                  {selectedItemData.details.map((detail, index) => (
                    <motion.li
                      key={index}
                      className="text-gray-300 flex items-start"
                      initial={{ opacity: 0, x: -20 }}
                      animate={{ opacity: 1, x: 0 }}
                      transition={{ delay: index * 0.1 }}
                    >
                      <div className="w-2 h-2 bg-cyan-400 rounded-full mr-3 mt-2 flex-shrink-0" />
                      {detail}
                    </motion.li>
                  ))}
                </ul>
              </div>
              
              <button
                className="w-full bg-gradient-to-r from-cyan-500 to-blue-600 text-white py-3 rounded-xl font-semibold hover:from-cyan-600 hover:to-blue-700 transition-all transform hover:scale-105"
                style={{
                  boxShadow: '0 4px 20px rgba(0, 240, 255, 0.3)',
                }}
              >
                Launch {selectedItemData.name.split(' ')[1]}
              </button>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>

      {/* Footer */}
      <AnimatePresence>
        {animationPhase === 'menu' && (
          <motion.div
            className="absolute bottom-8 left-1/2 -translate-x-1/2 text-cyan-400 text-2xl font-mono tracking-widest"
            style={{ textShadow: '0 0 20px #00f0ff' }}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 1, duration: 0.8 }}
          >
            KNIRV.COM
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};

export default App;