import { create } from "zustand";

interface AudioState {
  backgroundMusic: HTMLAudioElement | null;
  hitSound: HTMLAudioElement | null;
  successSound: HTMLAudioElement | null;
  isMuted: boolean;
  isInitialized: boolean;
  
  // Setter functions
  setBackgroundMusic: (music: HTMLAudioElement) => void;
  setHitSound: (sound: HTMLAudioElement) => void;
  setSuccessSound: (sound: HTMLAudioElement) => void;
  setInitialized: (initialized: boolean) => void;
  
  // Control functions
  toggleMute: () => void;
  playHit: () => void;
  playSuccess: () => void;
  playOrbPulse: () => void;
  playElectricShock: () => void;
  playExpansion: () => void;
  playClick: () => void;
  startAmbientMusic: () => void;
  stopAmbientMusic: () => void;
}

export const useAudio = create<AudioState>((set, get) => ({
  backgroundMusic: null,
  hitSound: null,
  successSound: null,
  isMuted: false, // Start unmuted for debugging
  isInitialized: false,
  
  setBackgroundMusic: (music) => set({ backgroundMusic: music }),
  setHitSound: (sound) => set({ hitSound: sound }),
  setSuccessSound: (sound) => set({ successSound: sound }),
  setInitialized: (initialized) => set({ isInitialized: initialized }),
  
  toggleMute: () => {
    const { isMuted } = get();
    const newMutedState = !isMuted;
    
    // Just update the muted state
    set({ isMuted: newMutedState });
    
    // Log the change
    console.log(`Audio ${newMutedState ? 'muted' : 'unmuted'}`);
  },
  
  playHit: () => {
    const { hitSound, isMuted } = get();
    if (hitSound) {
      // If sound is muted, don't play anything
      if (isMuted) {
        console.log("Hit sound skipped (muted)");
        return;
      }
      
      // Clone the sound to allow overlapping playback
      const soundClone = hitSound.cloneNode() as HTMLAudioElement;
      soundClone.volume = 0.3;
      soundClone.play().catch(error => {
        console.log("Hit sound play prevented:", error);
      });
    }
  },
  
  playSuccess: () => {
    const { successSound, isMuted } = get();
    if (successSound && !isMuted) {
      successSound.currentTime = 0;
      successSound.volume = 0.4;
      successSound.play().catch(error => {
        console.log("Success sound play prevented:", error);
      });
    }
  },

  playOrbPulse: () => {
    const { hitSound, isMuted } = get();
    if (hitSound && !isMuted) {
      const soundClone = hitSound.cloneNode() as HTMLAudioElement;
      soundClone.volume = 0.1;
      soundClone.playbackRate = 0.8;
      soundClone.play().catch(error => {
        console.log("Orb pulse sound play prevented:", error);
      });
    }
  },

  playElectricShock: () => {
    const { hitSound, isMuted } = get();
    if (hitSound && !isMuted) {
      const soundClone = hitSound.cloneNode() as HTMLAudioElement;
      soundClone.volume = 0.15;
      soundClone.playbackRate = 1.5;
      soundClone.play().catch(error => {
        console.log("Electric shock sound play prevented:", error);
      });
    }
  },

  playExpansion: () => {
    const { successSound, isMuted } = get();
    if (successSound && !isMuted) {
      successSound.currentTime = 0;
      successSound.volume = 0.3;
      successSound.playbackRate = 0.7;
      successSound.play().catch(error => {
        console.log("Expansion sound play prevented:", error);
      });
    }
  },

  playClick: () => {
    const { hitSound, isMuted } = get();
    if (hitSound && !isMuted) {
      const soundClone = hitSound.cloneNode() as HTMLAudioElement;
      soundClone.volume = 0.2;
      soundClone.playbackRate = 1.2;
      soundClone.play().catch(error => {
        console.log("Click sound play prevented:", error);
      });
    }
  },

  startAmbientMusic: () => {
    const { backgroundMusic, isMuted } = get();
    if (backgroundMusic && !isMuted) {
      backgroundMusic.volume = 0.1;
      backgroundMusic.loop = true;
      backgroundMusic.play().catch(error => {
        console.log("Ambient music play prevented:", error);
      });
    }
  },

  stopAmbientMusic: () => {
    const { backgroundMusic } = get();
    if (backgroundMusic) {
      backgroundMusic.pause();
      backgroundMusic.currentTime = 0;
    }
  }
}));
