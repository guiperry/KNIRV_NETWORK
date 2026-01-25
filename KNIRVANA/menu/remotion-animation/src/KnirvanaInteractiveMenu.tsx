import React, { useState, useEffect } from 'react';
import { AbsoluteFill, interpolate, useCurrentFrame, Sequence } from 'remotion';
import { Star, Zap, Cpu, Network, Shield, Wallet, Router, Server, Database, Globe } from 'lucide-react';

interface MenuItem {
  id: string;
  name: string;
  icon: React.ReactNode;
  description: string;
  angle: number;
  radius: number;
  ring: 'inner' | 'outer';
  color: string;
}

const KnirvanaInteractiveMenu: React.FC = () => {
  const frame = useCurrentFrame();
  const [hoveredItem, setHoveredItem] = useState<string | null>(null);
  const [selectedItem, setSelectedItem] = useState<string | null>(null);
  const [mousePos, setMousePos] = useState({ x: 0, y: 0 });

  const duration = 300;
  const menuStartFrame = 150;

  const starsOpacity = interpolate(frame, [0, 60], [0, 1]);
  const supernovaScale = interpolate(frame, [60, 90], [0.1, 3]);
  const siriusScale = interpolate(frame, [90, 120, 150], [0, 1.2, 0.8]);
  const siriusRotation = interpolate(frame, [90, 150], [0, 360]);
  const siriusCoreOpacity = interpolate(frame, [120, 150], [1, 0.3]);
  const electricSparksOpacity = interpolate(frame, [120, 150], [0, 1]);
  const menuScale = interpolate(frame, [menuStartFrame, 220], [0, 1]);
  const menuOpacity = interpolate(frame, [menuStartFrame, 180], [0, 1]);
  const iconSpread = interpolate(frame, [menuStartFrame, 220], [0, 1]);
  const lingeringFlicker = interpolate(frame, [250, 300], [1, 0.8]);

  const menuItems: MenuItem[] = [
    { id: 'pay', name: 'KNIRV PAY', icon: <Wallet size={32} />, description: 'Payment Gateway', angle: 0, radius: 180, ring: 'inner', color: '#ffffff' },
    { id: 'chain', name: 'KNIRV CHAIN', icon: <Database size={32} />, description: 'Blockchain Layer', angle: 180, radius: 180, ring: 'inner', color: '#ffffff' },
    { id: 'nexus', name: 'KNIRV NEXUS', icon: <Network size={32} />, description: 'Distributed Validation', angle: 270, radius: 180, ring: 'inner', color: '#ffffff' },
    { id: 'oracle', name: 'KNIRV ORACLE', icon: <Server size={32} />, description: 'Cross-Chain Hub', angle: 90, radius: 180, ring: 'inner', color: '#ffffff' },
    { id: 'gateway', name: 'KNIRV GATEWAY', icon: <Globe size={32} />, description: 'API Gateway', angle: 45, radius: 280, ring: 'outer', color: '#00f0ff' },
    { id: 'wallet', name: 'KNIRV WALLET', icon: <Wallet size={32} />, description: 'Non-Custodial Wallet', angle: 225, radius: 280, ring: 'outer', color: '#00f0ff' },
    { id: 'router', name: 'KNIRV ROUTER', icon: <Router size={32} />, description: 'P2P Network', angle: 315, radius: 280, ring: 'outer', color: '#00f0ff' },
    { id: 'controller', name: 'KNIRV CONTROLLER', icon: <Cpu size={32} />, description: 'AI Gateway', angle: 135, radius: 280, ring: 'outer', color: '#00f0ff' }
  ];

  const getItemPosition = (item: MenuItem) => {
    const radian = ((item.angle - 90) * Math.PI) / 180;
    const x = 960 + Math.cos(radian) * item.radius * iconSpread;
    const y = 540 + Math.sin(radian) * item.radius * iconSpread;
    return { x, y };
  };

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      setMousePos({ x: e.clientX, y: e.clientY });
    };
    window.addEventListener('mousemove', handleMouseMove);
    return () => window.removeEventListener('mousemove', handleMouseMove);
  }, []);

  return (
    <AbsoluteFill style={{ backgroundColor: '#040c1c', overflow: 'hidden', cursor: 'crosshair' }}>
      {/* Stars Background */}
      <div style={{ opacity: starsOpacity, position: 'absolute', width: '100%', height: '100%' }}>
        {Array.from({ length: 150 }).map((_, i) => (
          <Star
            key={i}
            size={Math.random() * 6 + 1}
            color="#a0d8ef"
            style={{
              position: 'absolute',
              top: `${Math.random() * 100}%`,
              left: `${Math.random() * 100}%`,
              opacity: Math.random() * 0.8 + 0.2,
              animation: `twinkle ${2 + Math.random() * 3}s infinite`,
            }}
          />
        ))}
      </div>

      {/* Supernova */}
      <Sequence from={60}>
        <div
          style={{
            position: 'absolute',
            top: '50%',
            left: '50%',
            transform: `translate(-50%, -50%) scale(${supernovaScale})`,
            opacity: interpolate(frame, [85, 95], [1, 0]),
          }}
        >
          <div
            style={{
              width: '300px',
              height: '300px',
              borderRadius: '50%',
              background: 'radial-gradient(circle, #ffffff, #00aaff, transparent)',
              boxShadow: '0 0 150px 75px #00aaff',
            }}
          />
        </div>
      </Sequence>

      {/* Sirius Star */}
      <Sequence from={90}>
        <div
          style={{
            position: 'absolute',
            top: '50%',
            left: '50%',
            transform: `translate(-50%, -50%) scale(${siriusScale}) rotate(${siriusRotation}deg)`,
          }}
        >
          <div
            style={{
              width: '180px',
              height: '180px',
              borderRadius: '50%',
              border: '6px solid #00f0ff',
              background: `radial-gradient(circle, rgba(0, 240, 255, ${siriusCoreOpacity}), rgba(0, 20, 60, 1))`,
              boxShadow: '0 0 60px 30px #00f0ff, inset 0 0 40px 15px #00f0ff',
              display: 'flex',
              justifyContent: 'center',
              alignItems: 'center',
            }}
          >
            <div style={{ opacity: electricSparksOpacity }}>
              <Zap color="#00f0ff" size={50} style={{ position: 'absolute', top: '-25px', left: '50%', transform: 'translateX(-50%)' }} />
              <Zap color="#00f0ff" size={50} style={{ position: 'absolute', bottom: '-25px', left: '50%', transform: 'translateX(-50%) rotate(180deg)' }} />
              <Zap color="#00f0ff" size={40} style={{ position: 'absolute', left: '-20px', top: '50%', transform: 'translateY(-50%) rotate(270deg)' }} />
              <Zap color="#00f0ff" size={40} style={{ position: 'absolute', right: '-20px', top: '50%', transform: 'translateY(-50%) rotate(90deg)' }} />
            </div>
          </div>
        </div>
      </Sequence>

      {/* Interactive Menu */}
      <Sequence from={menuStartFrame}>
        <div
          style={{
            position: 'absolute',
            top: '50%',
            left: '50%',
            transform: `translate(-50%, -50%) scale(${menuScale})`,
            opacity: menuOpacity,
          }}
        >
          {/* Central Title */}
          <h1 
            style={{ 
              color: '#ffffff', 
              fontSize: '3.5em', 
              textAlign: 'center', 
              textShadow: '0 0 30px #00f0ff',
              marginBottom: '20px',
              letterSpacing: '0.1em',
              fontFamily: 'monospace'
            }}
          >
            KNIRVANA
          </h1>

          {/* Menu Items */}
          {menuItems.map((item) => {
            const position = getItemPosition(item);
            const isHovered = hoveredItem === item.id;
            const isSelected = selectedItem === item.id;
            const scale = isHovered ? 1.2 : isSelected ? 1.1 : 1;
            
            return (
              <div
                key={item.id}
                style={{
                  position: 'absolute',
                  left: `${position.x - 60}px`,
                  top: `${position.y - 60}px`,
                  width: '120px',
                  height: '120px',
                  transform: `scale(${scale})`,
                  transition: 'all 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
                  cursor: 'pointer',
                  zIndex: isHovered ? 100 : 1,
                }}
                onMouseEnter={() => setHoveredItem(item.id)}
                onMouseLeave={() => setHoveredItem(null)}
                onClick={() => setSelectedItem(item.id)}
              >
                {/* Orb Container */}
                <div
                  style={{
                    width: '100%',
                    height: '100%',
                    borderRadius: '50%',
                    background: isHovered 
                      ? `radial-gradient(circle, ${item.color}22, ${item.color}11)`
                      : `radial-gradient(circle, ${item.color}11, transparent)`,
                    border: `2px solid ${item.color}`,
                    boxShadow: isHovered 
                      ? `0 0 40px 20px ${item.color}88, inset 0 0 20px ${item.color}44`
                      : `0 0 20px 10px ${item.color}44, inset 0 0 10px ${item.color}22`,
                    display: 'flex',
                    flexDirection: 'column',
                    justifyContent: 'center',
                    alignItems: 'center',
                    backdropFilter: 'blur(10px)',
                    position: 'relative',
                    overflow: 'hidden',
                  }}
                >
                  {/* Animated Ring */}
                  <div
                    style={{
                      position: 'absolute',
                      width: '100%',
                      height: '100%',
                      borderRadius: '50%',
                      border: `1px solid ${item.color}`,
                      opacity: isHovered ? 0.8 : 0.4,
                      animation: isHovered ? 'pulse 1s infinite' : 'none',
                    }}
                  />
                  
                  {/* Icon */}
                  <div style={{ color: item.color, marginBottom: '8px', zIndex: 2 }}>
                    {item.icon}
                  </div>
                  
                  {/* Name */}
                  <div
                    style={{
                      color: item.color,
                      fontSize: '0.7em',
                      fontWeight: 'bold',
                      textAlign: 'center',
                      textShadow: `0 0 10px ${item.color}`,
                      zIndex: 2,
                    }}
                  >
                    {item.name.split(' ')[1]}
                  </div>
                </div>
                
                {/* Hover Description */}
                {isHovered && (
                  <div
                    style={{
                      position: 'absolute',
                      top: '-40px',
                      left: '50%',
                      transform: 'translateX(-50%)',
                      background: 'rgba(0, 0, 0, 0.9)',
                      color: item.color,
                      padding: '8px 16px',
                      borderRadius: '20px',
                      fontSize: '0.8em',
                      whiteSpace: 'nowrap',
                      border: `1px solid ${item.color}`,
                      boxShadow: `0 4px 20px ${item.color}66`,
                      zIndex: 101,
                    }}
                  >
                    {item.description}
                  </div>
                )}
              </div>
            );
          })}

          {/* Constellation Lines */}
          <svg
            width="1920"
            height="1080"
            style={{
              position: 'absolute',
              top: '50%',
              left: '50%',
              transform: 'translate(-50%, -50%)',
              opacity: lingeringFlicker,
              pointerEvents: 'none',
            }}
          >
            {/* Orbital Rings */}
            <circle cx="960" cy="540" r={180 * iconSpread} stroke="#00f0ff" strokeWidth="2" fill="none" strokeDasharray="10,5" opacity="0.6" />
            <circle cx="960" cy="540" r={280 * iconSpread} stroke="#00f0ff" strokeWidth="2" fill="none" strokeDasharray="10,5" opacity="0.4" />
            
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

          {/* Selected Item Display */}
          {selectedItem && (
            <div
              style={{
                position: 'absolute',
                bottom: '-150px',
                left: '50%',
                transform: 'translateX(-50%)',
                background: 'rgba(0, 0, 0, 0.8)',
                border: '2px solid #00f0ff',
                borderRadius: '20px',
                padding: '20px 40px',
                color: '#ffffff',
                textAlign: 'center',
                boxShadow: '0 0 30px #00f0ff',
                backdropFilter: 'blur(10px)',
              }}
            >
              <div style={{ fontSize: '1.5em', fontWeight: 'bold', marginBottom: '10px' }}>
                {menuItems.find(item => item.id === selectedItem)?.name}
              </div>
              <div style={{ fontSize: '1em', color: '#00f0ff' }}>
                {menuItems.find(item => item.id === selectedItem)?.description}
              </div>
            </div>
          )}
        </div>
      </Sequence>

      {/* Footer */}
      <div 
        style={{ 
          position: 'absolute', 
          bottom: '30px', 
          width: '100%', 
          textAlign: 'center', 
          color: '#00f0ff', 
          fontSize: '2em', 
          opacity: menuOpacity,
          fontFamily: 'monospace',
          letterSpacing: '0.2em',
          textShadow: '0 0 20px #00f0ff'
        }}
      >
        KNIRV.COM
      </div>

      {/* CSS Animations */}
      <style>{`
        @keyframes twinkle {
          0%, 100% { opacity: 0.2; }
          50% { opacity: 1; }
        }
        @keyframes pulse {
          0%, 100% { transform: scale(1); opacity: 0.4; }
          50% { transform: scale(1.1); opacity: 0.8; }
        }
      `}</style>
    </AbsoluteFill>
  );
};

export default KnirvanaInteractiveMenu;