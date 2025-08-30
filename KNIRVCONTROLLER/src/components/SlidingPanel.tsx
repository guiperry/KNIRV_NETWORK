import React, { useEffect, useRef } from 'react';
import { X } from 'lucide-react';
interface SlidingPanelProps {
  id: string;
  isOpen: boolean;
  onClose: () => void;
  title: string;
  side: 'left' | 'right';
  children: React.ReactNode;
}

export const SlidingPanel: React.FC<SlidingPanelProps> = ({
  id: _id,
  isOpen,
  onClose,
  title,
  side,
  children
}) => {
  const panelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (panelRef.current && !panelRef.current.contains(event.target as Node)) {
        onClose();
      }
    };

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/30 backdrop-blur-sm transition-opacity duration-300 z-40"
      />

      {/* Panel */}
      <div
        ref={panelRef}
        className={`fixed top-0 ${side === 'left' ? 'left-0' : 'right-0'} h-full w-96 bg-gray-900/95 backdrop-blur-sm border-${side === 'left' ? 'r' : 'l'} border-gray-700/50 z-50 transition-transform duration-300 ease-in-out transform translate-x-0`}
      >
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-gray-700/50">
          <h2 className="text-lg font-semibold text-white">{title}</h2>
          <button
            onClick={onClose}
            className="w-8 h-8 flex items-center justify-center rounded-lg bg-gray-700/80 hover:bg-gray-600/80 transition-colors border border-gray-600/50"
          >
            <X className="w-4 h-4 text-gray-300 hover:text-white" />
          </button>
        </div>
        
        {/* Content */}
        <div className="p-4 overflow-y-auto h-full pb-20">
          {children}
        </div>
      </div>
    </>
  );
};