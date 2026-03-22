# Quick Start Guide

## Option 1: Browser Version (Easiest - No Installation)

Just open `standalone.html` in your browser:

1. Open the file in Chrome/Edge/Firefox
2. Press F11 for fullscreen mode for best experience
3. The center area simulates where your desktop would show through

**Note**: The browser version simulates the transparent effect but cannot actually show your desktop through it.

## Option 2: Electron Version (True Transparency)

This version creates a truly transparent window showing your desktop through the center.

### Prerequisites

- Node.js (v16 or higher)
- npm (comes with Node.js)

### Installation

```bash
# Navigate to project folder
cd transparent-dashboard

# Install dependencies
npm install

# Build TypeScript files
npm run build

# Run the application
npm start
```

### First Time Setup

```bash
# If you don't have Node.js, download it from: https://nodejs.org/

# Clone or extract the project files
cd transparent-dashboard

# Install all dependencies (this may take a few minutes)
npm install

# Start the dashboard
npm start
```

## Features Comparison

| Feature | Browser Version | Electron Version |
|---------|----------------|------------------|
| Easy to run | ✅ Yes | ⚠️ Requires setup |
| True transparency | ❌ No | ✅ Yes |
| Desktop pass-through | ❌ No | ✅ Yes |
| Real system metrics | ❌ Simulated | ✅ Real |
| Always on top | ❌ No | ✅ Yes |

## Customization

### Change Colors

Edit `styles.css` or the `<style>` section in `standalone.html`:

- Cyan theme: `#00ffff` (current)
- Green theme: Change to `#00ff00`
- Purple theme: Change to `#ff00ff`
- Orange theme: Change to `#ff8800`

### Adjust Panel Sizes

In CSS, modify:
- `.left-panel { width: 250px; }` - Left panel width
- `.right-panel { width: 250px; }` - Right panel width
- `.top-panel { height: 60px; }` - Top panel height
- `.bottom-panel { height: 60px; }` - Bottom panel height

### Update Frequency

In `renderer.js` or `<script>` section:
```javascript
setInterval(updateMetrics, 2000); // Change 2000 to desired milliseconds
```

## Troubleshooting

### Electron won't start
- Make sure Node.js is installed: `node --version`
- Try: `npm install electron --save-dev`
- Try: `npm rebuild`

### Transparency not working (Electron)
- **Windows**: Should work out of the box
- **macOS**: Should work out of the box
- **Linux**: Requires compositor (compton, picom, or built-in)

### Browser version shows errors
- Use a modern browser (Chrome, Edge, Firefox)
- Some features may not work in older browsers

## Advanced Configuration

### Make center area click-through

In Electron `main.ts`, add after creating window:
```typescript
mainWindow.setIgnoreMouseEvents(true, { forward: true });
```

### Change window size

In `main.ts`:
```typescript
const { width, height } = screen.getPrimaryDisplay().workAreaSize;
mainWindow = new BrowserWindow({
    width: width * 0.8,  // 80% of screen width
    height: height * 0.8, // 80% of screen height
    // ...
});
```

### Disable always on top

In `main.ts`, change:
```typescript
alwaysOnTop: false,  // Changed from true
```

## Project Structure

```
transparent-dashboard/
├── package.json          # Node.js dependencies
├── tsconfig.json         # TypeScript configuration
├── README.md            # This file
├── index.html           # Main HTML (Electron)
├── styles.css           # Stylesheet (Electron)
├── renderer.js          # UI logic (Electron)
├── standalone.html      # Browser version (all-in-one)
└── src/
    └── main.ts          # Electron main process
```

## Tips

1. **Fullscreen mode**: Press F11 in browser version for immersive experience
2. **Move window**: Drag the top panel (Electron version)
3. **Development**: Use `npm run dev` to rebuild and run
4. **Debugging**: Uncomment `mainWindow.webContents.openDevTools()` in main.ts

## System Requirements

- **Browser version**: Any modern browser
- **Electron version**: 
  - Windows 10/11
  - macOS 10.13+
  - Linux with compositor support
  - 2GB RAM minimum
  - 200MB disk space

## Performance

- CPU usage: ~1-2% (Electron)
- Memory: ~100-150MB (Electron)
- No significant impact on system performance
