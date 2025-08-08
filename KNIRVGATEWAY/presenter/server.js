const express = require('express');
const path = require('path');
const fs = require('fs');

const app = express();
const PORT = 3000;

// Enable CORS for all routes
app.use((req, res, next) => {
    res.header('Access-Control-Allow-Origin', '*');
    res.header('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS');
    res.header('Access-Control-Allow-Headers', 'Origin, X-Requested-With, Content-Type, Accept');
    next();
});

// Serve static files from current directory
app.use(express.static(__dirname, {
    setHeaders: (res, path) => {
        // Set proper MIME types
        if (path.endsWith('.html')) {
            res.setHeader('Content-Type', 'text/html; charset=utf-8');
        } else if (path.endsWith('.css')) {
            res.setHeader('Content-Type', 'text/css');
        } else if (path.endsWith('.js')) {
            res.setHeader('Content-Type', 'application/javascript');
        }
        
        // Disable caching for development
        res.setHeader('Cache-Control', 'no-cache, no-store, must-revalidate');
        res.setHeader('Pragma', 'no-cache');
        res.setHeader('Expires', '0');
    }
}));

// API endpoint to get presentation info
app.get('/api/presentations', (req, res) => {
    const presentations = {};

    // Scan for presentation directories in the presentations folder
    const presentationsPath = path.join(__dirname, 'presentations');

    if (fs.existsSync(presentationsPath)) {
        const presentationDirs = fs.readdirSync(presentationsPath).filter(item => {
            return fs.statSync(path.join(presentationsPath, item)).isDirectory();
        });

        presentationDirs.forEach(dir => {
            const dirPath = path.join(presentationsPath, dir);
            const files = fs.readdirSync(dirPath);
            const slideFiles = files.filter(file => file.match(/^Slide_\d+\.html$/));
            presentations[dir] = {
                name: dir.replace(/([A-Z])/g, ' $1').trim(),
                slides: slideFiles.length
            };
        });
    }

    res.json(presentations);
});

// API endpoint to check if a slide exists
app.get('/api/slide-exists/:presentation/:slide', (req, res) => {
    const { presentation, slide } = req.params;
    const slidePath = path.join(__dirname, 'presentations', presentation, `Slide_${slide}.html`);

    res.json({
        exists: fs.existsSync(slidePath),
        path: `presentations/${presentation}/Slide_${slide}.html`
    });
});

// Fallback route for SPA
app.get('*', (req, res) => {
    // If it's a slide request, try to serve it from presentations folder
    if (req.path.includes('Slide_') && req.path.endsWith('.html')) {
        // Handle both direct paths and presentations/ prefixed paths
        let filePath;
        if (req.path.startsWith('/presentations/')) {
            filePath = path.join(__dirname, req.path);
        } else {
            // Assume it's a direct presentation/slide path
            filePath = path.join(__dirname, 'presentations', req.path);
        }

        if (fs.existsSync(filePath)) {
            res.sendFile(filePath);
        } else {
            res.status(404).send(`
                <!DOCTYPE html>
                <html>
                <head>
                    <title>Slide Not Found</title>
                    <style>
                        body {
                            font-family: Arial, sans-serif;
                            text-align: center;
                            padding: 50px;
                            background: #0a0e27;
                            color: white;
                        }
                        .error { color: #ef4444; }
                    </style>
                </head>
                <body>
                    <h1 class="error">Slide Not Found</h1>
                    <p>The requested slide does not exist:</p>
                    <code>${req.path}</code>
                </body>
                </html>
            `);
        }
    } else {
        // Try to serve index.html for other routes
        const indexPath = path.join(__dirname, 'index.html');
        if (fs.existsSync(indexPath)) {
            res.sendFile(indexPath);
        } else {
            res.status(404).send('File not found');
        }
    }
});

app.listen(PORT, () => {
    console.log(`🚀 KNIRV Presentation Server running at http://localhost:${PORT}`);
    console.log(`📊 Slider: http://localhost:${PORT}/presentation-slider.html`);
    console.log(`🏠 Index: http://localhost:${PORT}/index.html`);
    console.log(`📁 Serving from: ${__dirname}`);
});
