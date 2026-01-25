import { useEffect } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from './ui/card';
import { Button } from './ui/button';
import { X } from 'lucide-react';
import { useAudio } from '../lib/stores/useAudio';

interface ServiceModalProps {
  service: string | null;
  isOpen: boolean;
  onClose: () => void;
}

const serviceInfo: Record<string, { title: string; description: string; url: string }> = {
  'KNIRVANA': {
    title: 'KNIRVANA Core',
    description: 'The central hub of the KNIRV Network ecosystem.',
    url: 'https://knirvana.com'
  },
  'KNIRVTESTNET': {
    title: 'KNIRV Testnet',
    description: 'Development and testing environment for KNIRV applications.',
    url: 'https://testnet.knirv.com'
  },
  'KNIRVSDK': {
    title: 'KNIRV SDK',
    description: 'Software Development Kit for building on KNIRV Network.',
    url: 'https://sdk.knirv.com'
  },
  'KNIRVROUTER': {
    title: 'KNIRV Router',
    description: 'Intelligent routing system for KNIRV Network traffic.',
    url: 'https://router.knirv.com'
  },
  'KNIRVORACLE': {
    title: 'KNIRV Oracle',
    description: 'Decentralized oracle network for external data feeds.',
    url: 'https://oracle.knirv.com'
  },
  'KNIRVNEXUS': {
    title: 'KNIRV Nexus',
    description: 'Central connection point for all KNIRV services.',
    url: 'https://nexus.knirv.com'
  },
  'KNIRVGRAPH': {
    title: 'KNIRV Graph',
    description: 'Data visualization and analytics platform.',
    url: 'https://graph.knirv.com'
  },
  'KNIRVGATEWAY': {
    title: 'KNIRV Gateway',
    description: 'API gateway for KNIRV Network services.',
    url: 'https://gateway.knirv.com'
  },
  'KNIRVCORTEX': {
    title: 'KNIRV Cortex',
    description: 'AI-powered decision making engine.',
    url: 'https://cortex.knirv.com'
  },
  'KNIRVCONTROLLER': {
    title: 'KNIRV Controller',
    description: 'Network management and control interface.',
    url: 'https://controller.knirv.com'
  },
  'KNIRVCLI': {
    title: 'KNIRV CLI',
    description: 'Command-line interface for KNIRV Network.',
    url: 'https://cli.knirv.com'
  },
  'KNIRVCHAIN': {
    title: 'KNIRV Chain',
    description: 'Blockchain infrastructure for the KNIRV ecosystem.',
    url: 'https://chain.knirv.com'
  }
};

const ServiceModal = ({ service, isOpen, onClose }: ServiceModalProps) => {
  const { playSuccess, playClick } = useAudio();
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      }
    };

    if (isOpen) {
      document.addEventListener('keydown', handleEscape);
      // Play success sound when modal opens
      playSuccess();
    }

    return () => {
      document.removeEventListener('keydown', handleEscape);
    };
  }, [isOpen, onClose, playSuccess]);

  if (!isOpen || !service) return null;

  const info = serviceInfo[service];

  return (
    <div 
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm"
      onClick={onClose}
    >
      <Card 
        className="w-full max-w-2xl mx-4 bg-gray-900/95 border-cyan-500/30 text-white"
        onClick={(e) => e.stopPropagation()}
      >
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="text-cyan-400 text-xl font-bold">
            {info?.title || service}
          </CardTitle>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => {
              playClick();
              onClose();
            }}
            className="text-gray-400 hover:text-white"
          >
            <X className="h-4 w-4" />
          </Button>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-gray-300">
            {info?.description || 'Service information not available.'}
          </p>
          
          {info?.url && (
            <div className="border border-cyan-500/20 rounded-lg overflow-hidden">
              <iframe
                src={info.url}
                className="w-full h-64"
                title={`${service} Interface`}
                sandbox="allow-scripts allow-same-origin"
                onError={() => console.log(`Failed to load iframe for ${service}`)}
              />
            </div>
          )}
          
          <div className="flex justify-end space-x-2">
            <Button
              variant="outline"
              onClick={() => {
                playClick();
                onClose();
              }}
              className="border-cyan-500/30 text-cyan-400 hover:bg-cyan-500/10"
            >
              Close
            </Button>
            {info?.url && (
              <Button
                onClick={() => {
                  playClick();
                  window.open(info.url, '_blank');
                }}
                className="bg-cyan-500 hover:bg-cyan-600 text-black font-semibold"
              >
                Open in New Tab
              </Button>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
};

export default ServiceModal;
