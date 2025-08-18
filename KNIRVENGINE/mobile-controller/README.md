KNIRVAGENTIFIER

### 🤖 KNIRV-AGENTIFIER: The Autonomous Gateway
Technology: Rust WASM-powered AI agents with SEAL loop + Advanced Voice Integration

Purpose: A mobile-native adapter that empowers existing AI assistants with autonomous agentic abilities, acting as the primary user gateway to the knirv.com D-TEN.

## 🎯 Key Features

### Core Functionality
- CodeT5 Base LLM + personalized LoRA adapters
- Continuous failure detection and solution proposal
- User Delegation Certificate (UDC) orchestration
- Skill invocation and NRN consumption
- Seamless integration with KNIRV-WALLET for asset management and transaction execution
- Secure communication channels between the agentifier and the D-TEN

### 🎤 Voice Integration & Monitoring
- **Advanced Voice Processing**: Real-time speech recognition with Web Speech API
- **Cognitive Shell**: Intelligent voice command parsing and execution
- **Edge Coloring System**: Visual feedback for voice status (listening, processing, speaking)
- **Wake Word Detection**: "KNIRV" wake word for hands-free activation
- **Multi-Modal Commands**: Support for navigation, skill activation, and system control
- **Real-time Status Monitoring**: Visual indicators for voice activity and cognitive mode

### 🎨 Visual Feedback System
- **Dynamic Edge Coloring**: Screen borders change color based on voice status
  - 🟢 Green: Idle state
  - 🔵 Teal: Listening for commands
  - 🔵 Blue: Processing voice input
  - 🟣 Purple: Speaking/responding
  - 🔴 Red: Error state
- **Intensity Modulation**: Edge brightness reflects activity level
- **Smooth Transitions**: 500ms animated color transitions

## 🗣️ Voice Commands

### Navigation Commands
- "Show skills page" / "Navigate to skills"
- "Open wallet" / "Navigate to wallet"
- "Show UDC" / "Open certificate panel"
- "Go home" / "Show agents"

### System Commands
- "Toggle cognitive mode" / "Enable advanced mode"
- "Check agent status" / "Show agent health"
- "Check NRN balance" / "Show balance"
- "Show network status" / "Check connections"

### Skill Commands
- "Activate skill [name]" / "Enable skill [name]"
- "Deactivate skill [name]" / "Disable skill [name]"
- "Show available skills"

## 🏗️ Architecture

### Voice Processing Pipeline
1. **Audio Capture**: MediaRecorder API for audio input
2. **Speech Recognition**: Web Speech API with continuous listening
3. **Command Parsing**: Pattern matching for command extraction
4. **Action Execution**: Navigation and system control
5. **Visual Feedback**: Edge coloring and status updates
6. **Speech Synthesis**: Text-to-speech responses

### Component Structure
```
src/
├── shared/
│   └── cognitive-shell/
│       ├── EventEmitter.ts      # Browser-compatible event system
│       └── VoiceProcessor.ts     # Core voice processing logic
├── react-app/
│   ├── components/
│   │   ├── EdgeColoring.tsx     # Visual feedback system
│   │   ├── VoiceControl.tsx     # Voice interface component
│   │   └── Layout.tsx           # Main layout with voice integration
│   └── hooks/
│       └── useVoiceIntegration.ts # Voice state management hook
```

## 🚀 Getting Started

### Prerequisites
- Node.js 20+ (for development)
- Modern browser with Web Speech API support
- Microphone access for voice features

### Installation
```bash
npm install
npm run dev
```

### Voice Feature Testing
Open `test-voice-integration.html` in your browser to test voice functionality:
```bash
# Open the test file directly in browser
open test-voice-integration.html
```

### Usage
1. **Enable Voice**: Click the microphone button in the bottom-right corner
2. **Speak Commands**: Use any of the supported voice commands
3. **Visual Feedback**: Watch the screen edges change color based on voice status
4. **Cognitive Mode**: Toggle advanced voice processing features

## 🔧 Configuration

### Voice Processor Settings
```typescript
const config: VoiceConfig = {
  sampleRate: 16000,
  channels: 1,
  language: 'en-US',
  enableWakeWord: true,
  wakeWord: 'knirv',
  noiseReduction: true,
};
```

### Edge Coloring Customization
```typescript
const edgeColors = {
  idle: '#10B981',      // Green
  listening: '#14B8A6', // Teal
  processing: '#3B82F6', // Blue
  speaking: '#8B5CF6',   // Purple
  error: '#EF4444'       // Red
};
```

## 🧪 Testing

The application includes comprehensive voice integration testing:
- Real-time speech recognition
- Command pattern matching
- Visual feedback systems
- Error handling and recovery
- Cross-browser compatibility

## 🔮 Future Enhancements

- **Multi-language Support**: Expand beyond English
- **Custom Wake Words**: User-configurable activation phrases
- **Voice Biometrics**: Speaker identification and authentication
- **Offline Processing**: Local speech recognition capabilities
- **Advanced NLP**: Context-aware command interpretation
- **Voice Shortcuts**: Customizable voice macros

## 📱 Mobile Compatibility

The voice integration is designed for mobile-first usage:
- Touch-friendly voice controls
- Responsive edge coloring
- Optimized for mobile browsers
- Progressive Web App (PWA) ready

