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
      console.log('AudioManager: Initializing audio assets...');
      
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
        console.log(`AudioManager: Sound loaded (${loadedCount}/${totalSounds})`);
        if (loadedCount === totalSounds) {
          console.log('All audio assets loaded successfully');
          setInitialized(true);
          
          // Don't auto-start music, let user interaction handle it
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