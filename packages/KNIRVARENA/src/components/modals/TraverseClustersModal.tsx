import React, { useEffect, useRef } from 'react';

interface TraverseClustersModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export const TraverseClustersModal: React.FC<TraverseClustersModalProps> = ({
  isOpen,
  onClose
}) => {
  const iframeRef = useRef<HTMLIFrameElement>(null);

  useEffect(() => {
    const handleMessage = (event: MessageEvent) => {
      if (event.data && event.data.type === 'CLOSE_TRAVERSE_CLUSTERS') {
        onClose();
      }
    };

    window.addEventListener('message', handleMessage);
    return () => window.removeEventListener('message', handleMessage);
  }, [onClose]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-90 flex items-center justify-center z-50">
      <div className="w-full h-full max-w-[1200px] max-h-[800px] relative">
        <button
          onClick={onClose}
          className="absolute top-4 right-4 z-10 bg-cyan-500 text-black px-4 py-2 rounded font-bold hover:bg-cyan-400"
        >
          ← Return to Arena
        </button>
        <iframe
          ref={iframeRef}
          src="/flythrough.html"
          className="w-full h-full border-2 border-cyan-500"
          title="Traverse Clusters"
        />
      </div>
    </div>
  );
};