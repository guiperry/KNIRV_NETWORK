import { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';

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

  // Update edge color based on voice status
  useEffect(() => {
    const getEdgeColor = () => {
      switch (state.voiceStatus) {
        case 'listening': return '#14B8A6'; // Teal
        case 'processing': return '#3B82F6'; // Blue
        case 'speaking': return '#8B5CF6'; // Purple
        case 'error': return '#EF4444'; // Red
        default: return '#10B981'; // Green
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

    // Navigation commands
    if (lowerCommand.includes('skills') || lowerCommand.includes('skill')) {
      navigate('/skills');
    } else if (lowerCommand.includes('wallet')) {
      navigate('/wallet');
    } else if (lowerCommand.includes('udc') || lowerCommand.includes('certificate')) {
      navigate('/udc');
    } else if (lowerCommand.includes('home') || lowerCommand.includes('agents')) {
      navigate('/');
    }

    // Voice mode commands
    else if (lowerCommand.includes('cognitive mode') || lowerCommand.includes('advanced mode')) {
      setState(prev => ({ ...prev, cognitiveMode: !prev.cognitiveMode }));
    }

    // Skill-specific commands
    else if (lowerCommand.includes('activate skill') || lowerCommand.includes('deactivate skill')) {
      console.log('Skill activation requested:', command);
      console.log('Skill deactivation requested:', command);
    }

    // Agent commands
    else if (lowerCommand.includes('check agent') || lowerCommand.includes('agent status')) {
      console.log('Agent status check requested');
    } else if (lowerCommand.includes('deploy agent')) {
      console.log('Agent deployment requested');
    }

    // System commands
    else if (lowerCommand.includes('status') || lowerCommand.includes('health')) {
      console.log('System status check requested');
    } else if (lowerCommand.includes('balance') || lowerCommand.includes('nrn')) {
      console.log('NRN balance check requested');
    }

    // Simulate processing time and voice response
    setTimeout(() => {
      setState(prev => ({ ...prev, voiceStatus: 'speaking' }));

      // Simulate speaking response
      setTimeout(() => {
        setState(prev => ({ ...prev, voiceStatus: 'idle' }));
      }, 1000);
    }, 800);
  }, [navigate]);

  const toggleVoice = useCallback((active: boolean) => {
    setState(prev => ({
      ...prev,
      isVoiceActive: active,
      voiceStatus: active ? 'listening' : 'idle'
    }));
  }, []);

  const toggleCognitiveMode = useCallback(() => {
    setState(prev => ({ ...prev, cognitiveMode: !prev.cognitiveMode }));
  }, []);

  const setVoiceStatus = useCallback((status: VoiceStatus) => {
    setState(prev => ({ ...prev, voiceStatus: status }));
  }, []);

  // Simulate voice response with edge coloring
  const speakResponse = useCallback(async (text: string) => {
    setState(prev => ({ ...prev, voiceStatus: 'speaking' }));
    
    // Simulate speech duration
    const duration = Math.max(2000, text.length * 50);
    
    setTimeout(() => {
      setState(prev => ({ ...prev, voiceStatus: 'idle' }));
    }, duration);
  }, []);

  return {
    // State
    isVoiceActive: state.isVoiceActive,
    voiceStatus: state.voiceStatus,
    cognitiveMode: state.cognitiveMode,
    lastCommand: state.lastCommand,
    edgeColor: state.edgeColor,
    edgeIntensity: state.edgeIntensity,
    
    // Actions
    handleVoiceCommand,
    toggleVoice,
    toggleCognitiveMode,
    setVoiceStatus,
    speakResponse
  };
};
