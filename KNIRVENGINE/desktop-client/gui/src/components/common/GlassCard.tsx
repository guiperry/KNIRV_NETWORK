import React from 'react';

interface GlassCardProps {
  children: React.ReactNode;
  title?: string;
  darker?: boolean;
  className?: string;
}

const GlassCard: React.FC<GlassCardProps> = ({ 
  children, 
  title, 
  darker = false, 
  className = '' 
}) => {
  return (
    <div className={`
      backdrop-blur-md rounded-2xl border border-slate-700/50 shadow-2xl p-6 mb-6
      ${darker 
        ? 'bg-slate-800/30' 
        : 'bg-slate-800/50'
      }
      ${className}
    `}>
      {title && (
        <h3 className="text-blue-400 text-lg font-semibold mb-4 mt-0">
          {title}
        </h3>
      )}
      {children}
    </div>
  );
};

export default GlassCard;
