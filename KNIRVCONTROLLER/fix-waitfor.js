import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const filePath = path.join(__dirname, 'tests/unit/components/SettingsPanel.test.tsx');

// Read the file
let content = fs.readFileSync(filePath, 'utf8');

// Replace patterns
content = content
  // Remove async from function declarations that only have waitFor
  .replace(/async \(\) => \{[\s\S]*?await waitFor\(\(\) => \{([\s\S]*?)\}\);[\s\S]*?\}/g, '() => {$1}')
  
  // Replace simple waitFor patterns
  .replace(/await waitFor\(\(\) => \{([\s\S]*?)\}\);/g, '$1')
  
  // Replace async function declarations that only use waitFor
  .replace(/it\('([^']+)', async \(\) => \{/g, "it('$1', () => {")
  .replace(/beforeEach\(async \(\) => \{/g, 'beforeEach(() => {')
  
  // Clean up extra whitespace
  .replace(/\n\s*\n\s*\n/g, '\n\n');

// Write the file back
fs.writeFileSync(filePath, content);

console.log('Fixed waitFor patterns in SettingsPanel.test.tsx');
