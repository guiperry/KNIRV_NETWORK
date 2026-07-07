import { VoiceProcessor, type VoiceConfig } from '@/shared/cognitive-shell/VoiceProcessor';
import { canUseBrowserApis } from './browserStorage';

export function createVoiceProcessor(config: VoiceConfig) {
  if (!canUseBrowserApis() && typeof window === 'undefined') {
    return null;
  }

  try {
    const processor = new VoiceProcessor(config);
    return processor.isSupported() ? processor : null;
  } catch {
    return null;
  }
}
