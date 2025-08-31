import * as React from 'react';

interface EdgeColoringProps {
  color: string;
  intensity: number;
}

export const EdgeColoring: React.FC<EdgeColoringProps> = ({ color, intensity }) => {
  const opacity = Math.min(1, Math.max(0, intensity));
  
  return (
    <div className="absolute inset-0 pointer-events-none z-50">
      {/* Top Edge */}
      <div 
        className="absolute top-0 left-0 right-0 h-1 transition-all duration-500"
        style={{ 
          background: `linear-gradient(to right, transparent, ${color}, transparent)`,
          opacity: opacity
        }}
      />
      
      {/* Right Edge */}
      <div 
        className="absolute top-0 right-0 bottom-0 w-1 transition-all duration-500"
        style={{ 
          background: `linear-gradient(to bottom, transparent, ${color}, transparent)`,
          opacity: opacity
        }}
      />
      
      {/* Bottom Edge */}
      <div 
        className="absolute bottom-0 left-0 right-0 h-1 transition-all duration-500"
        style={{ 
          background: `linear-gradient(to right, transparent, ${color}, transparent)`,
          opacity: opacity
        }}
      />
      
      {/* Left Edge */}
      <div 
        className="absolute top-0 left-0 bottom-0 w-1 transition-all duration-500"
        style={{ 
          background: `linear-gradient(to bottom, transparent, ${color}, transparent)`,
          opacity: opacity
        }}
      />
    </div>
  );
};