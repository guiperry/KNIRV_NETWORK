import React, { useEffect, useRef } from 'react';
import { X } from 'lucide-react';

interface SlideDownModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
}

export const SlideDownModal: React.FC<SlideDownModalProps> = ({
  isOpen,
  onClose,
  title,
  children
}) => {
  const modalRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (modalRef.current && !modalRef.current.contains(event.target as Node)) {
        onClose();
      }
    };

    const handleEscapeKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      }
    };

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      document.addEventListener('keydown', handleEscapeKey);
      document.body.style.overflow = 'hidden';
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleEscapeKey);
      document.body.style.overflow = 'unset';
    };
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  return (
    <>
      {/* Backdrop */}
      <div className="fixed inset-0 bg-black/30 backdrop-blur-sm transition-opacity duration-300 z-40" />

      {/* Modal Container */}
      <div className="fixed inset-0 flex items-center justify-center z-50 p-4">
        <div
          ref={modalRef}
          className="w-full max-w-[720px] max-h-[85vh] bg-gray-900/95 backdrop-blur-sm border border-gray-700/50 rounded-xl shadow-2xl transition-all duration-300 ease-in-out transform scale-100 opacity-100"
        >
          {/* Header */}
          <div className="flex items-center justify-between px-5 py-4 border-b border-gray-700/50">
            <h2 className="text-xl font-bold bg-gradient-to-r from-blue-400 to-cyan-400 bg-clip-text text-transparent">
              {title}
            </h2>
            <button
              onClick={onClose}
              className="w-7 h-7 flex items-center justify-center rounded-lg bg-gray-700/80 hover:bg-gray-600/80 transition-colors border border-gray-600/50"
            >
              <X className="w-3.5 h-3.5 text-gray-300 hover:text-white" />
            </button>
          </div>
          
          {/* Content */}
          <div className="p-5 overflow-y-auto max-h-[70vh]">
            {children}
          </div>
        </div>
      </div>
    </>
  );
};