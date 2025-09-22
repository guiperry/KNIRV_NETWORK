import { useEffect, useRef } from 'react';
import { useAudio } from '../lib/stores/useAudio';

const AudioManager = () => {
  const { 
    setBackgroundMusic, 
    setHitSound, 
    setSuccessSound, 
    setInitialized, 
    isInitialized,
    startAmbientMusic
  } = useAudio();
  
  const initRef = useRef<boolean>(false);

  useEffect(() => {
    if (!isInitialized && !initRef.current) {
      initRef.current = true;
      // Initialize all audio assets
      const backgroundMusic = new Audio('/sounds/background.mp3');
      const hitSound = new Audio('/sounds/hit.mp3');
      const successSound = new Audio('/sounds/success.mp3');

      // Preload the audio
      backgroundMusic.preload = 'auto';
      hitSound.preload = 'auto';
      successSound.preload = 'auto';

      // Set up event listeners to know when sounds are ready
      let loadedCount = 0;
      const totalSounds = 3;

      const onSoundLoaded = () => {
        loadedCount++;
        if (loadedCount === totalSounds) {
          console.log('All audio assets loaded successfully');
          setInitialized(true);
          
          // Start ambient music if not muted (read current state from store)
          const currentState = useAudio.getState();
          if (!currentState.isMuted) {
            setTimeout(() => startAmbientMusic(), 1000);
          }
        }
      };

      backgroundMusic.addEventListener('loadeddata', onSoundLoaded);
      hitSound.addEventListener('loadeddata', onSoundLoaded);
      successSound.addEventListener('loadeddata', onSoundLoaded);

      // Error handling
      const onSoundError = (soundName: string) => (e: Event) => {
        console.warn(`Failed to load ${soundName}:`, e);
        onSoundLoaded(); // Still count as loaded to prevent blocking
      };

      backgroundMusic.addEventListener('error', onSoundError('background music'));
      hitSound.addEventListener('error', onSoundError('hit sound'));
      successSound.addEventListener('error', onSoundError('success sound'));

      // Store the audio elements
      setBackgroundMusic(backgroundMusic);
      setHitSound(hitSound);
      setSuccessSound(successSound);
    }
  }, [isInitialized, setBackgroundMusic, setHitSound, setSuccessSound, setInitialized, startAmbientMusic]);

  return null; // This component doesn't render anything
};

export default AudioManager;