# Qwen3-TTS Integration in KNIRVCONTROLLER as PWA

## Overview

This document outlines how to integrate Qwen3-TTS, a state-of-the-art text-to-speech (TTS) system developed by Qwen, into the KNIRVCONTROLLER application as a Progressive Web App (PWA). The integration will replace the existing Web Speech API-based speech synthesis with Qwen3-TTS, providing enhanced voice quality, multilingual support, and advanced voice control capabilities.

## Current Voice System Analysis

### KNIRVCONTROLLER Voice Architecture

The current voice system in KNIRVCONTROLLER uses the Web Speech API for:
- **Speech Recognition**: Browser's built-in SpeechRecognition API
- **Speech Synthesis**: Browser's built-in speechSynthesis API
- **Audio Processing**: Web Audio API for microphone input

Key components:
- [`VoiceControl.tsx`](./KNIRVCONTROLLER/src/components/VoiceControl.tsx) - UI component for voice interaction
- [`VoiceProcessor.ts`](./KNIRVCONTROLLER/src/sensory-shell/VoiceProcessor.ts) - Core voice processing engine

### Limitations of Current System

1. **Limited Voice Quality**: Browser speech synthesis engines vary in quality
2. **Limited Multilingual Support**: Depends on browser capabilities
3. **No Voice Customization**: Cannot create custom voices or control speaking style
4. **Inconsistent Performance**: Behavior differs across browsers and devices

## Qwen3-TTS Features

Qwen3-TTS offers significant improvements over the current system:

### Key Features

1. **High-Quality Speech Generation**: 1.7B parameter model with natural-sounding voices
2. **Multilingual Support**: 10+ languages (Chinese, English, Japanese, Korean, German, French, Russian, Portuguese, Spanish, Italian)
3. **Voice Customization**:
   - **Custom Voice**: 9 premium predefined voices with style control
   - **Voice Design**: Create custom voices via natural language descriptions
   - **Voice Clone**: Clone voices from 3-second audio samples
4. **Advanced Control**:
   - Speaking rate, pitch, volume control
   - Emotion and tone control via instructions
   - Style transfer capabilities
5. **Streaming Generation**: Low-latency streaming synthesis (97ms end-to-end)

### Available Models

| Model | Features | Language Support | Streaming | Instruction Control |
|-------|----------|------------------|-----------|---------------------|
| Qwen3-TTS-12Hz-1.7B-CustomVoice | 9 premium voices with style control | 10+ languages | ✅ | ✅ |
| Qwen3-TTS-12Hz-1.7B-VoiceDesign | Create voices via natural language | 10+ languages | ✅ | ✅ |
| Qwen3-TTS-12Hz-1.7B-Base | 3-second voice clone | 10+ languages | ✅ | |

## Integration Architecture

### High-Level Design

```
KNIRVCONTROLLER PWA
├── VoiceControl Component (UI)
│   └── VoiceProcessor (Core Engine)
│       ├── Speech Recognition (Web Speech API)
│       └── Speech Synthesis (Qwen3-TTS)
└── Qwen3-TTS Backend Service
    ├── Model Loading & Management
    └── TTS Generation Endpoints
```

### Integration Approaches

We will implement Qwen3-TTS integration using two main approaches:

1. **API-Based Integration**: Connect to a remote Qwen3-TTS service (DashScope API)
2. **Local Deployment**: Run Qwen3-TTS locally using vLLM or Hugging Face Transformers

## Implementation Steps

### 1. Backend Service Setup

#### Option A: DashScope API (Recommended for Production)

```bash
# Install required packages
pip install dashscope
```

Create a simple Flask or FastAPI backend service:

```python
from flask import Flask, request, jsonify
import dashscope
from dashscope import TextToSpeech

app = Flask(__name__)
dashscope.api_key = 'YOUR_API_KEY'

@app.route('/api/tts', methods=['POST'])
def text_to_speech():
    data = request.json
    text = data.get('text')
    language = data.get('language', 'English')
    speaker = data.get('speaker', 'Ryan')
    instruct = data.get('instruct', '')

    try:
        response = TextToSpeech.call(
            model='qwen3-tts-12hz-1.7b-customvoice',
            text=text,
            language=language,
            speaker=speaker,
            instruct=instruct
        )
        
        if response.status_code == 200:
            return jsonify({
                'audio_url': response.output['audio_url'],
                'audio_data': response.output['audio_data']
            })
        else:
            return jsonify({'error': 'TTS generation failed'}), 500
            
    except Exception as e:
        return jsonify({'error': str(e)}), 500

if __name__ == '__main__':
    app.run(debug=True, host='0.0.0.0', port=5000)
```

#### Option B: Local Deployment with Hugging Face

```python
import torch
import soundfile as sf
from qwen_tts import Qwen3TTSModel

# Load model (runs locally on GPU)
model = Qwen3TTSModel.from_pretrained(
    "Qwen/Qwen3-TTS-12Hz-1.7B-CustomVoice",
    device_map="cuda:0",
    dtype=torch.bfloat16,
    attn_implementation="flash_attention_2",
)

# Generate speech
wavs, sr = model.generate_custom_voice(
    text="Hello, this is Qwen3-TTS speaking!",
    language="English",
    speaker="Ryan",
    instruct="Speaking clearly and slowly"
)

# Save audio
sf.write("output.wav", wavs[0], sr)
```

### 2. Frontend Integration

#### Update VoiceProcessor.ts

Modify the `VoiceProcessor` class to use Qwen3-TTS instead of web speech synthesis:

```typescript
// src/sensory-shell/VoiceProcessor.ts

export class VoiceProcessor extends EventEmitter {
    // ... existing code ...
    
    private ttsEndpoint: string = '/api/tts'; // Your backend endpoint

    public async speak(text: string, options: unknown = {}): Promise<void> {
        const speechOptions = options as SpeechSynthesisOptions;
        
        try {
            const response = await fetch(this.ttsEndpoint, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    text: text,
                    language: speechOptions.language || this.config.language,
                    speaker: speechOptions.speaker || 'Ryan',
                    instruct: speechOptions.instruct || '',
                    rate: speechOptions.rate,
                    pitch: speechOptions.pitch,
                    volume: speechOptions.volume
                })
            });

            if (!response.ok) {
                throw new Error('TTS API request failed');
            }

            const data = await response.json();
            
            // Play audio
            const audioContext = new (window.AudioContext || (window as any).webkitAudioContext)();
            const audio = new Audio(data.audio_url);
            
            audio.onended = () => {
                this.emit('speechEnded', { text });
            };
            
            audio.onerror = (event) => {
                this.emit('speechError', event);
            };
            
            audio.onplay = () => {
                this.emit('speechStarted', { text });
            };
            
            await audio.play();
            
        } catch (error) {
            console.error('Qwen3-TTS API error:', error);
            
            // Fallback to web speech API if Qwen3-TTS fails
            if (this.synthesis) {
                const utterance = new SpeechSynthesisUtterance(text);
                // ... existing web speech API code ...
                this.synthesis?.speak(utterance);
            } else {
                throw new Error('Speech synthesis not available');
            }
        }
    }

    // ... existing code ...
}
```

#### Update VoiceControl.tsx

Add UI controls for Qwen3-TTS features:

```typescript
// src/components/VoiceControl.tsx

import React from 'react';
import { Mic, MicOff, Volume2, Brain, Settings } from 'lucide-react';
import { VoiceProcessor, VoiceConfig } from '../sensory-shell/VoiceProcessor';

interface VoiceControlProps {
  isActive: boolean;
  onVoiceCommand: (command: string) => void;
  onToggle: (active: boolean) => void;
  cognitiveMode?: boolean;
}

export const VoiceControl: React.FC<VoiceControlProps> = ({
  isActive,
  onVoiceCommand,
  onToggle,
  cognitiveMode = false
}) => {
  const [isListening, setIsListening] = useState(false);
  const [transcript, setTranscript] = useState('');
  const [confidence, setConfidence] = useState(0);
  const [voiceProcessor, setVoiceProcessor] = useState<VoiceProcessor | null>(null);
  const [isSupported, setIsSupported] = useState(false);
  const [selectedSpeaker, setSelectedSpeaker] = useState('Ryan');
  const [speakingRate, setSpeakingRate] = useState(1.0);
  const [pitch, setPitch] = useState(1.0);
  const [volume, setVolume] = useState(1.0);
  const [showSettings, setShowSettings] = useState(false);
  
  // Supported speakers for Qwen3-TTS CustomVoice model
  const supportedSpeakers = [
    { name: 'Vivian', description: 'Bright young female (Chinese)' },
    { name: 'Serena', description: 'Warm gentle female (Chinese)' },
    { name: 'Uncle_Fu', description: 'Seasoned male (Chinese)' },
    { name: 'Dylan', description: 'Beijing male dialect' },
    { name: 'Eric', description: 'Chengdu male dialect' },
    { name: 'Ryan', description: 'Dynamic male (English)' },
    { name: 'Aiden', description: 'Sunny American male' },
    { name: 'Ono_Anna', description: 'Playful Japanese female' },
    { name: 'Sohee', description: 'Warm Korean female' }
  ];

  // ... existing code ...

  const handleSpeakTest = async () => {
    if (voiceProcessor) {
      try {
        await voiceProcessor.speak('Hello, this is a test of Qwen3-TTS', {
          language: 'English',
          speaker: selectedSpeaker,
          rate: speakingRate,
          pitch: pitch,
          volume: volume
        });
      } catch (error) {
        console.error('Speech test failed:', error);
      }
    }
  };

  return (
    <div className="absolute bottom-4 right-4 z-40" data-testid="voice-control">
      <div className="flex flex-col items-end space-y-2">
        {/* Voice Settings */}
        {showSettings && (
          <div className="bg-gray-800/90 backdrop-blur-sm rounded-lg p-4 max-w-sm border border-gray-700/50">
            <div className="flex justify-between items-center mb-3">
              <h3 className="text-sm font-medium text-white">Voice Settings</h3>
              <button
                onClick={() => setShowSettings(false)}
                className="text-gray-400 hover:text-white"
              >
                ×
              </button>
            </div>
            
            {/* Speaker Selection */}
            <div className="mb-3">
              <label className="block text-xs text-gray-400 mb-1">Speaker</label>
              <select
                value={selectedSpeaker}
                onChange={(e) => setSelectedSpeaker(e.target.value)}
                className="w-full bg-gray-700/50 border border-gray-600 rounded-md text-sm text-white px-2 py-1"
              >
                {supportedSpeakers.map(speaker => (
                  <option key={speaker.name} value={speaker.name}>
                    {speaker.description}
                  </option>
                ))}
              </select>
            </div>

            {/* Speaking Rate */}
            <div className="mb-3">
              <label className="block text-xs text-gray-400 mb-1">
                Speaking Rate: {speakingRate.toFixed(1)}x
              </label>
              <input
                type="range"
                min="0.5"
                max="2"
                step="0.1"
                value={speakingRate}
                onChange={(e) => setSpeakingRate(parseFloat(e.target.value))}
                className="w-full"
              />
            </div>

            {/* Pitch */}
            <div className="mb-3">
              <label className="block text-xs text-gray-400 mb-1">
                Pitch: {pitch.toFixed(1)}x
              </label>
              <input
                type="range"
                min="0.5"
                max="2"
                step="0.1"
                value={pitch}
                onChange={(e) => setPitch(parseFloat(e.target.value))}
                className="w-full"
              />
            </div>

            {/* Volume */}
            <div className="mb-3">
              <label className="block text-xs text-gray-400 mb-1">
                Volume: {Math.round(volume * 100)}%
              </label>
              <input
                type="range"
                min="0"
                max="1"
                step="0.1"
                value={volume}
                onChange={(e) => setVolume(parseFloat(e.target.value))}
                className="w-full"
              />
            </div>

            {/* Test Button */}
            <button
              onClick={handleSpeakTest}
              className="w-full bg-teal-500 hover:bg-teal-600 text-white text-sm py-2 rounded-md transition-colors"
            >
              Test Voice
            </button>
          </div>
        )}

        {/* Voice Transcript */}
        {(isListening || transcript) && (
          <div className="bg-gray-800/90 backdrop-blur-sm rounded-lg p-3 max-w-xs border border-gray-700/50">
            {/* ... existing transcript display ... */}
          </div>
        )}

        {/* Voice Control Buttons */}
        <div className="flex items-center space-x-2">
          {/* Settings Button */}
          <button
            onClick={() => setShowSettings(!showSettings)}
            className="w-10 h-10 rounded-full bg-gray-700 text-gray-400 hover:bg-gray-600 flex items-center justify-center"
            data-testid="voice-control-settings"
          >
            <Settings className="w-5 h-5" />
          </button>

          {/* Toggle Button */}
          <button
            onClick={() => onToggle(!isActive)}
            className={`w-14 h-14 rounded-full flex items-center justify-center transition-all duration-300 relative ${
              isActive
                ? 'bg-teal-500 text-white shadow-lg shadow-teal-500/30'
                : 'bg-gray-700 text-gray-400 hover:bg-gray-600'
            }`}
            data-testid="voice-control-toggle"
          >
            {isActive ? (
              <Mic className="w-6 h-6" />
            ) : (
              <MicOff className="w-6 h-6" />
            )}

            {/* Cognitive Mode Indicator */}
            {cognitiveMode && (
              <div className="absolute -top-1 -right-1 w-4 h-4 bg-purple-500 rounded-full flex items-center justify-center">
                <Brain className="w-2 h-2 text-white" />
              </div>
            )}

            {/* Unsupported Indicator */}
            {cognitiveMode && !isSupported && (
              <div className="absolute -bottom-1 -right-1 w-3 h-3 bg-red-500 rounded-full"></div>
            )}
          </button>
        </div>
      </div>
    </div>
  );
};
```

### 3. PWA Service Worker Setup

Update the service worker to handle offline audio caching:

```javascript
// public/sw.js

// Cache Qwen3-TTS audio responses
self.addEventListener('fetch', (event) => {
  if (event.request.url.includes('/api/tts')) {
    event.respondWith(
      caches.open('qwen3-tts-cache').then((cache) => {
        return cache.match(event.request).then((response) => {
          const fetchPromise = fetch(event.request).then((networkResponse) => {
            cache.put(event.request, networkResponse.clone());
            return networkResponse;
          });
          return response || fetchPromise;
        });
      })
    );
  }
});

// Cache audio files
self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open('qwen3-tts-cache').then((cache) => {
      return cache.addAll([
        '/api/tts'
      ]);
    })
  );
});
```

### 4. Configuration and Environment Variables

Create a `.env` file for configuration:

```env
# Qwen3-TTS Configuration
QWEN3_TTS_API_KEY=your-api-key-here
QWEN3_TTS_ENDPOINT=https://dashscope.aliyuncs.com/api/v1
QWEN3_TTS_MODEL=qwen3-tts-12hz-1.7b-customvoice
QWEN3_TTS_DEFAULT_SPEAKER=Ryan
QWEN3_TTS_DEFAULT_LANGUAGE=en-US
QWEN3_TTS_DEFAULT_RATE=1.0
QWEN3_TTS_DEFAULT_PITCH=1.0
QWEN3_TTS_DEFAULT_VOLUME=1.0
```

## Deployment Options

### 1. Production Deployment (Cloud)

- **Backend**: Deploy Python backend to AWS Lambda, GCP Cloud Functions, or Azure Functions
- **API Gateway**: Use AWS API Gateway or Cloudflare Workers for API management
- **Frontend**: Deploy PWA to Vercel, Netlify, or Cloudflare Pages
- **Caching**: Use CDN for audio file caching
- **Authentication**: Implement API key management

### 2. Local Development

```bash
# Run backend service
python backend/app.py

# Run frontend development server
cd KNIRVCONTROLLER
npm run dev
```

### 3. Edge Deployment (Vercel Edge Functions)

```typescript
// api/tts/route.ts
import { TextToSpeech } from 'dashscope';

export async function POST(request: Request) {
  const { text, language = 'English', speaker = 'Ryan', instruct = '' } = await request.json();
  
  try {
    const response = await TextToSpeech.call({
      model: 'qwen3-tts-12hz-1.7b-customvoice',
      text,
      language,
      speaker,
      instruct
    });

    return new Response(JSON.stringify(response), {
      headers: { 'Content-Type': 'application/json' }
    });
  } catch (error) {
    return new Response(JSON.stringify({ error: error.message }), {
      status: 500,
      headers: { 'Content-Type': 'application/json' }
    });
  }
}
```

## Performance Optimization

### 1. Audio Caching

- Cache frequently used phrases
- Implement LRU cache for audio responses
- Pre-cache common system messages

### 2. Streaming Audio

- Use Web Audio API for streaming playback
- Implement audio chunking for long messages
- Optimize for low-latency delivery

### 3. Offline Support

- Cache generated audio files in service worker
- Implement offline fallback to web speech API
- Pre-load default voice samples

## Testing Strategy

### Unit Tests

```typescript
// tests/unit/VoiceProcessor.test.ts

import { VoiceProcessor } from '../src/sensory-shell/VoiceProcessor';

describe('VoiceProcessor Qwen3-TTS Integration', () => {
  let voiceProcessor: VoiceProcessor;

  beforeEach(() => {
    voiceProcessor = new VoiceProcessor();
  });

  it('should initialize with Qwen3-TTS support', async () => {
    const isSupported = voiceProcessor.isSupported();
    expect(isSupported).toBe(true);
  });

  it('should generate speech using Qwen3-TTS API', async () => {
    const text = 'Test speech generation';
    const spy = jest.spyOn(global.fetch, 'bind(global)');
    
    await voiceProcessor.speak(text);
    
    expect(spy).toHaveBeenCalled();
    expect(spy.mock.calls[0][0]).toContain('/api/tts');
    expect(spy.mock.calls[0][1]?.body).toContain(text);
  });

  it('should fallback to web speech API if Qwen3-TTS fails', async () => {
    const text = 'Fallback test';
    const errorSpy = jest.spyOn(console, 'error');
    
    // Mock fetch failure
    jest.spyOn(global, 'fetch').mockRejectedValue(new Error('API failure'));
    
    await voiceProcessor.speak(text);
    
    expect(errorSpy).toHaveBeenCalled();
    expect(errorSpy.mock.calls[0][0]).toContain('Qwen3-TTS API error');
  });
});
```

### Integration Tests

```typescript
// tests/integration/Qwen3TTSIntegration.test.ts

describe('Qwen3-TTS API Integration', () => {
  it('should respond with valid audio URL', async () => {
    const response = await fetch('/api/tts', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        text: 'Hello, world!',
        language: 'English',
        speaker: 'Ryan'
      })
    });

    expect(response.ok).toBe(true);
    const data = await response.json();
    expect(data.audio_url).toBeDefined();
    expect(typeof data.audio_url).toBe('string');
  });

  it('should support different speakers', async () => {
    const speakers = ['Vivian', 'Ryan', 'Aiden', 'Ono_Anna'];
    
    for (const speaker of speakers) {
      const response = await fetch('/api/tts', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          text: 'Test speaker: ' + speaker,
          language: 'English',
          speaker: speaker
        })
      });

      expect(response.ok).toBe(true);
    }
  });
});
```

## Security Considerations

### API Key Protection

- Store API keys in environment variables
- Implement rate limiting
- Use API key rotation
- Consider JWT authentication for user-specific keys

### Data Privacy

- Redact sensitive information from TTS requests
- Implement audio file expiration
- Use HTTPS for all API calls
- Provide data deletion capabilities

### Input Validation

- Sanitize text inputs to prevent injection attacks
- Validate speaker and language parameters
- Implement rate limiting per user

## Future Enhancements

### 1. Advanced Voice Features

- Voice clone functionality from user-recorded audio
- Voice design via natural language descriptions
- Real-time voice style transfer
- Emotion detection and adaptive synthesis

### 2. Performance Improvements

- WebAssembly optimization for local model deployment
- PWA background processing
- Edge computing for low-latency responses
- Audio compression and format optimization

### 3. Integration with Cognitive System

- Voice control of cognitive mode
- Context-aware speech synthesis
- Multimodal integration with visual and textual inputs
- Adaptive speaking styles based on user context

### 4. Accessibility

- ARIA labels for voice controls
- Keyboard shortcuts for voice operations
- Text-to-speech fallback for screen readers
- Support for high-contrast mode

## Conclusion

Integrating Qwen3-TTS into KNIRVCONTROLLER as a PWA will significantly enhance the voice interaction capabilities, providing high-quality, customizable speech synthesis that surpasses the limitations of browser-built-in engines. The implementation supports both cloud-based API integration for production and local deployment for development, ensuring flexibility and scalability.

Key benefits include:
1. Professional-quality speech generation
2. Multilingual and dialect support
3. Customizable voices and speaking styles
4. Low-latency streaming synthesis
5. Offline capabilities through service workers
6. Enhanced accessibility features

This integration will elevate the user experience of KNIRVCONTROLLER, making voice interactions more natural, engaging, and effective.
