import { Volume2, VolumeX } from 'lucide-react';
import { Button } from './ui/button';
import { useAudio } from '../lib/stores/useAudio';

const AudioControls = () => {
  const { isMuted, toggleMute, startAmbientMusic, stopAmbientMusic, isInitialized } = useAudio();

  const handleToggle = () => {
    if (!isInitialized) return;
    
    toggleMute();
    
    // Manage ambient music based on mute state
    if (isMuted) {
      // Currently muted, will become unmuted
      setTimeout(() => startAmbientMusic(), 100);
    } else {
      // Currently unmuted, will become muted
      stopAmbientMusic();
    }
  };

  return (
    <div className="fixed top-4 right-4 z-50">
      <Button
        onClick={handleToggle}
        variant="ghost"
        size="icon"
        className="bg-black/30 hover:bg-black/50 text-cyan-400 hover:text-cyan-300 border border-cyan-500/20 backdrop-blur-sm"
        title={isMuted ? "Unmute Audio" : "Mute Audio"}
        disabled={!isInitialized}
      >
        {isMuted ? (
          <VolumeX className="h-5 w-5" />
        ) : (
          <Volume2 className="h-5 w-5" />
        )}
      </Button>
    </div>
  );
};

export default AudioControls;