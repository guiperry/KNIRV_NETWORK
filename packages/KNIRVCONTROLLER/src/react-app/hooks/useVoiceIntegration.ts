import { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router';
import { Capacitor } from '@capacitor/core';
import { startNativeListening, stopNativeListening, speakNative } from '@/react-app/platform/voiceBridge';

export type VoiceStatus = 'idle' | 'listening' | 'processing' | 'speaking' | 'error';

interface VoiceIntegrationState {
  isVoiceActive: boolean;
  voiceStatus: VoiceStatus;
  cognitiveMode: boolean;
  lastCommand: string;
  edgeColor: string;
  edgeIntensity: number;
}

export const useVoiceIntegration = () => {
  const navigate = useNavigate();

  const [state, setState] = useState<VoiceIntegrationState>({
    isVoiceActive: false,
    voiceStatus: 'idle',
    cognitiveMode: false,
    lastCommand: '',
    edgeColor: '#10B981',
    edgeIntensity: 0.3
  });

  useEffect(() => {
    const getEdgeColor = () => {
      switch (state.voiceStatus) {
        case 'listening': return '#14B8A6';
        case 'processing': return '#3B82F6';
        case 'speaking': return '#8B5CF6';
        case 'error': return '#EF4444';
        default: return '#10B981';
      }
    };

    const getEdgeIntensity = () => {
      switch (state.voiceStatus) {
        case 'listening': return 0.8;
        case 'processing': return 0.9;
        case 'speaking': return 1.0;
        case 'error': return 0.7;
        default: return state.isVoiceActive ? 0.5 : 0.3;
      }
    };

    setState(prev => ({
      ...prev,
      edgeColor: getEdgeColor(),
      edgeIntensity: getEdgeIntensity()
    }));
  }, [state.voiceStatus, state.isVoiceActive]);

  const handleVoiceCommand = useCallback((command: string) => {
    setState(prev => ({ ...prev, lastCommand: command, voiceStatus: 'processing' }));

    const lowerCommand = command.toLowerCase();

    if (lowerCommand.includes('badges') || lowerCommand.includes('badge')) {
      navigate('/badges');
    } else if (lowerCommand.includes('vault')) {
      navigate('/vault');
    } else if (lowerCommand.includes('udc') || lowerCommand.includes('certificate')) {
      navigate('/udc');
    } else if (lowerCommand.includes('scan') || lowerCommand.includes('scanner')) {
      navigate('/scanner');
    } else if (lowerCommand.includes('workflow')) {
      navigate('/workflows');
    } else if (lowerCommand.includes('home') || lowerCommand.includes('verifier') || lowerCommand.includes('node')) {
      navigate('/dves');
    } else if (lowerCommand.includes('cognitive mode') || lowerCommand.includes('advanced mode')) {
      setState(prev => ({ ...prev, cognitiveMode: !prev.cognitiveMode }));
    } else if (lowerCommand.includes('balance') || lowerCommand.includes('how much')) {
       console.log('Checking balance...');
    } else if (lowerCommand.includes('send') || lowerCommand.includes('transfer')) {
       navigate('/vault');
       console.log('Initiating transfer...');
    } else if (lowerCommand.includes('activate badge')) {
      console.log('Badge activation requested:', command);
    } else if (lowerCommand.includes('deactivate badge')) {
      console.log('Badge deactivation requested:', command);
    } else if (lowerCommand.includes('check workflow') || lowerCommand.includes('workflow status')) {
      console.log('Workflow status check requested');
    } else if (lowerCommand.includes('deploy workflow')) {
      console.log('Workflow deployment requested');
    } else if (lowerCommand.includes('status') || lowerCommand.includes('health')) {
      console.log('System status check requested');
    }

    setTimeout(() => {
      setState(prev => ({ ...prev, voiceStatus: 'speaking' }));
      setTimeout(() => {
        setState(prev => ({ ...prev, voiceStatus: 'idle' }));
      }, 1000);
    }, 800);
  }, [navigate]);

  const toggleVoice = useCallback(async (active: boolean) => {
    setState(prev => ({
      ...prev,
      isVoiceActive: active,
      voiceStatus: active ? 'listening' : 'idle'
    }));

    if (Capacitor.isNativePlatform()) {
      if (active) {
        try {
          await startNativeListening(handleVoiceCommand);
        } catch (err) {
          console.error('Failed to start native voice:', err);
          setState(prev => ({ ...prev, isVoiceActive: false, voiceStatus: 'error' }));
        }
      } else {
        await stopNativeListening();
      }
    }
  }, [handleVoiceCommand]);

  const toggleCognitiveMode = useCallback(() => {
    setState(prev => ({ ...prev, cognitiveMode: !prev.cognitiveMode }));
  }, []);

  const setVoiceStatus = useCallback((status: VoiceStatus) => {
    setState(prev => ({ ...prev, voiceStatus: status }));
  }, []);

  const speakResponse = useCallback(async (text: string) => {
    setState(prev => ({ ...prev, voiceStatus: 'speaking' }));

    if (Capacitor.isNativePlatform()) {
      try {
        await speakNative(text);
      } catch (err) {
        console.error('Failed to speak native:', err);
      }
      setState(prev => ({ ...prev, voiceStatus: 'idle' }));
      return;
    }

    const duration = Math.max(2000, text.length * 50);
    setTimeout(() => {
      setState(prev => ({ ...prev, voiceStatus: 'idle' }));
    }, duration);
  }, []);

  return {
    isVoiceActive: state.isVoiceActive,
    voiceStatus: state.voiceStatus,
    cognitiveMode: state.cognitiveMode,
    lastCommand: state.lastCommand,
    edgeColor: state.edgeColor,
    edgeIntensity: state.edgeIntensity,
    handleVoiceCommand,
    toggleVoice,
    toggleCognitiveMode,
    setVoiceStatus,
    speakResponse
  };
};
