import { Capacitor } from '@capacitor/core';
import { SpeechRecognition } from '@capgo/capacitor-speech-recognition';
import { TextToSpeech } from '@capacitor-community/text-to-speech';

export async function startNativeListening(onResult: (text: string) => void): Promise<void> {
  if (!Capacitor.isNativePlatform()) {
    throw new Error('Native voice bridge is only available on native platforms.');
  }

  const { speechRecognition } = await SpeechRecognition.requestPermissions();
  if (speechRecognition !== 'granted') throw new Error('Microphone permission denied');

  await SpeechRecognition.start({ language: 'en-US', maxResults: 1, partialResults: true, popup: false });

  await SpeechRecognition.addListener('partialResults', (data: { matches?: string[] }) => {
    if (data.matches?.[0]) onResult(data.matches[0]);
  });
}

export async function stopNativeListening(): Promise<void> {
  if (!Capacitor.isNativePlatform()) return;
  await SpeechRecognition.stop();
  await SpeechRecognition.removeAllListeners();
}

export async function speakNative(text: string): Promise<void> {
  if (!Capacitor.isNativePlatform()) {
    throw new Error('Native voice bridge is only available on native platforms.');
  }
  await TextToSpeech.speak({ text, lang: 'en-US', rate: 0.9, pitch: 1.0, volume: 1.0 });
}
