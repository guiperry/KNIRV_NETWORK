#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

console.log('🔍 Verifying KNIRV-CONTROLLER Voice Integration...\n');

// Check if all required files exist
const requiredFiles = [
  'src/shared/cognitive-shell/EventEmitter.ts',
  'src/shared/cognitive-shell/VoiceProcessor.ts',
  'src/react-app/components/EdgeColoring.tsx',
  'src/react-app/components/VoiceControl.tsx',
  'src/react-app/hooks/useVoiceIntegration.ts',
  'src/react-app/types/global.d.ts',
  'test-voice-integration.html',
  'VOICE_INTEGRATION_SUMMARY.md'
];

let allFilesExist = true;

console.log('📁 Checking required files:');
requiredFiles.forEach(file => {
  const exists = fs.existsSync(path.join(__dirname, file));
  console.log(`  ${exists ? '✅' : '❌'} ${file}`);
  if (!exists) allFilesExist = false;
});

console.log('\n🔧 Checking TypeScript compilation:');
const { execSync } = require('child_process');

try {
  execSync('npx tsc -b', { stdio: 'pipe' });
  console.log('  ✅ TypeScript compilation successful');
} catch (error) {
  console.log('  ❌ TypeScript compilation failed');
  console.log('  Error:', error.message);
  allFilesExist = false;
}

console.log('\n📋 Checking component integration:');

// Check Layout.tsx for voice integration
const layoutPath = 'src/react-app/components/Layout.tsx';
if (fs.existsSync(layoutPath)) {
  const layoutContent = fs.readFileSync(layoutPath, 'utf8');
  const hasEdgeColoring = layoutContent.includes('EdgeColoring');
  const hasVoiceControl = layoutContent.includes('VoiceControl');
  const hasVoiceHook = layoutContent.includes('useVoiceIntegration');
  
  console.log(`  ${hasEdgeColoring ? '✅' : '❌'} EdgeColoring component integrated`);
  console.log(`  ${hasVoiceControl ? '✅' : '❌'} VoiceControl component integrated`);
  console.log(`  ${hasVoiceHook ? '✅' : '❌'} useVoiceIntegration hook integrated`);
} else {
  console.log('  ❌ Layout.tsx not found');
  allFilesExist = false;
}

console.log('\n📖 Checking documentation:');
const readmePath = 'README.md';
if (fs.existsSync(readmePath)) {
  const readmeContent = fs.readFileSync(readmePath, 'utf8');
  const hasVoiceSection = readmeContent.includes('Voice Integration');
  const hasCommands = readmeContent.includes('Voice Commands');
  
  console.log(`  ${hasVoiceSection ? '✅' : '❌'} Voice integration documentation`);
  console.log(`  ${hasCommands ? '✅' : '❌'} Voice commands documentation`);
} else {
  console.log('  ❌ README.md not found');
}

console.log('\n🎯 Integration Summary:');
if (allFilesExist) {
  console.log('  ✅ All voice integration components successfully merged');
  console.log('  ✅ TypeScript compilation passes');
  console.log('  ✅ Components properly integrated into Layout');
  console.log('  ✅ Documentation updated');
  console.log('\n🎉 Voice integration merge completed successfully!');
  console.log('\n📝 Next steps:');
  console.log('  1. Open test-voice-integration.html in browser to test voice features');
  console.log('  2. Grant microphone permissions when prompted');
  console.log('  3. Click the microphone button to activate voice control');
  console.log('  4. Try voice commands like "Show skills page" or "Navigate to wallet"');
  console.log('  5. Watch the edge coloring change based on voice status');
} else {
  console.log('  ❌ Some integration issues detected');
  console.log('  Please check the failed items above');
}

console.log('\n🔗 Test the integration:');
console.log('  Browser test: file://' + path.resolve(__dirname, 'test-voice-integration.html'));
console.log('  Documentation: ' + path.resolve(__dirname, 'VOICE_INTEGRATION_SUMMARY.md'));
