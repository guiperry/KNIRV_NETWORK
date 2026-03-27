# Transparent Desktop

A futuristic HUD-style transparent desktop built with Electron and TypeScript, featuring:

- **Transparent window** - Desktop shows through the center
- **Real-time system monitoring** - CPU, memory, network, disk I/O
- **Interactive panels** - Around all edges (top, bottom, left, right)
- **Performance charts** - Live visualization of system metrics
- **Futuristic design** - Cyan glow effects, HUD-style interface

## Installation

```bash
cd transparent-dashboard
npm install
```

## Running the Desktop

```bash
npm start
```

Or for development:

```bash
npm run dev
```

## Features

### Panels
- **Top Panel**: System status, CPU/Memory usage, time, uptime
- **Left Panel**: System metrics with bars, quick stats (threads, temp, fan)
- **Right Panel**: Performance chart, notifications, system info
- **Bottom Panel**: Network and disk I/O stats, window controls

### Transparency
- The center area is fully transparent
- Your desktop shows through the middle
- Panels have semi-transparent dark backgrounds with blur effects

### Customization
You can customize:
- Colors in `styles.css` (search for `#00ffff` for cyan theme)
- Panel sizes and positions
- Update intervals in `renderer.js`
- Add more metrics or remove existing ones

## Technical Details

- **Electron**: For transparent window support
- **TypeScript**: For type-safe code
- **HTML5 Canvas**: For performance charts
- **Node.js OS module**: For real system metrics

## Controls

- **Minimize button** (_): Minimize the window
- **Close button** (X): Close the application
- **Drag**: Click and drag the top panel to move the window

## Platform Support

- **Windows**: Full support with transparency
- **macOS**: Full support with transparency
- **Linux**: Works with compositing window managers (requires compositor)

## Notes

- The window is set to "always on top" by default
- You can modify the transparency level in `styles.css` by changing the `rgba` values
- To make certain areas click-through, modify the `pointer-events` CSS property
- The desktop automatically scales to your screen size

## Troubleshooting

If transparency doesn't work:
- On Linux: Make sure you have a compositor running (like picom, compton, or built-in compositor)
- Check that your system supports window transparency
- Try adjusting the `rgba` alpha values in CSS

## License

MIT
