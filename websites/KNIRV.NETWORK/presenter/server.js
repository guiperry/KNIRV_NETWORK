const express = require('express');
const path = require('path');
const fs = require('fs');
const multer = require('multer');
const AdmZip = require('adm-zip');
const { execSync } = require('child_process');

const app = express();
const PORT = 3000;

// Configure multer for file uploads
const upload = multer({
    dest: 'uploads/',
    limits: { fileSize: 50 * 1024 * 1024 } // 50MB limit
});

// Enable CORS for all routes
app.use((req, res, next) => {
    res.header('Access-Control-Allow-Origin', '*');
    res.header('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS');
    res.header('Access-Control-Allow-Headers', 'Origin, X-Requested-With, Content-Type, Accept');
    next();
});

// Parse JSON bodies
app.use(express.json());

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

// API endpoint to scan for presentations without auth.json
app.get('/api/scan-presentations', (req, res) => {
    try {
        const presentationsPath = path.join(__dirname, 'presentations');
        const presentations = [];

        if (fs.existsSync(presentationsPath)) {
            const dirs = fs.readdirSync(presentationsPath).filter(item => {
                return fs.statSync(path.join(presentationsPath, item)).isDirectory();
            });

            dirs.forEach(dir => {
                const dirPath = path.join(presentationsPath, dir);
                const authPath = path.join(dirPath, 'auth.json');
                const hasAuth = fs.existsSync(authPath);

                // Count slides
                const files = fs.readdirSync(dirPath);
                const slideFiles = files.filter(file => file.match(/^Slide_\d+\.html$/));

                if (slideFiles.length > 0) {
                    presentations.push({
                        folder: dir,
                        name: dir.replace(/([A-Z])/g, ' $1').trim(),
                        slides: slideFiles.length,
                        hasAuth: hasAuth
                    });
                }
            });
        }

        res.json({ presentations });
    } catch (error) {
        console.error('Error scanning presentations:', error);
        res.status(500).json({ error: 'Failed to scan presentations' });
    }
});

// API endpoint to get all presentations with auth.json files
app.get('/api/presentations-with-auth', (req, res) => {
    try {
        const presentationsPath = path.join(__dirname, 'presentations');
        const presentations = {};

        if (fs.existsSync(presentationsPath)) {
            const dirs = fs.readdirSync(presentationsPath).filter(item => {
                return fs.statSync(path.join(presentationsPath, item)).isDirectory();
            });

            dirs.forEach(dir => {
                const dirPath = path.join(presentationsPath, dir);
                const authPath = path.join(dirPath, 'auth.json');

                if (fs.existsSync(authPath)) {
                    try {
                        const authData = JSON.parse(fs.readFileSync(authPath, 'utf-8'));

                        // Count slides
                        const files = fs.readdirSync(dirPath);
                        const slideFiles = files.filter(file => file.match(/^Slide_\d+\.html$/));

                        presentations[dir] = {
                            name: authData.presentationName,
                            description: authData.description,
                            slides: slideFiles.length,
                            password: authData.password
                        };
                    } catch (e) {
                        console.error(`Error reading auth.json for ${dir}:`, e);
                    }
                }
            });
        }

        res.json({ presentations });
    } catch (error) {
        console.error('Error loading presentations with auth:', error);
        res.status(500).json({ error: 'Failed to load presentations' });
    }
});

// Helper function to find slide files recursively
function findSlideFiles(dir) {
    const slideFiles = [];

    function searchDir(currentDir) {
        const items = fs.readdirSync(currentDir);

        items.forEach(item => {
            const itemPath = path.join(currentDir, item);
            const stat = fs.statSync(itemPath);

            if (stat.isDirectory()) {
                // Recursively search subdirectories
                searchDir(itemPath);
            } else if (item.match(/^Slide_\d+\.html$/i)) {
                // Found a slide file
                slideFiles.push(itemPath);
            }
        });
    }

    searchDir(dir);
    return slideFiles.sort(); // Sort to ensure proper order
}

// API endpoint to import a presentation
app.post('/api/import-presentation', upload.single('zipFile'), async (req, res) => {
    try {
        const { presentationName, password, description, folderName, type, existingFolder } = req.body;
        const presentationsPath = path.join(__dirname, 'presentations');
        const targetPath = path.join(presentationsPath, folderName);

        // Ensure presentations directory exists
        if (!fs.existsSync(presentationsPath)) {
            fs.mkdirSync(presentationsPath, { recursive: true });
        }

        let slideCount = 0;

        if (type === 'zip' && req.file) {
            // Handle zip file upload
            const zip = new AdmZip(req.file.path);

            // Create target directory
            if (!fs.existsSync(targetPath)) {
                fs.mkdirSync(targetPath, { recursive: true });
            }

            // Extract zip contents to a temporary directory first
            const tempExtractPath = path.join(__dirname, 'temp_extract_' + Date.now());
            fs.mkdirSync(tempExtractPath, { recursive: true });

            try {
                zip.extractAllTo(tempExtractPath, true);

                // Find all HTML slide files in the extracted content (recursively)
                const slideFiles = findSlideFiles(tempExtractPath);

                // Copy slide files to the target directory (flattened structure)
                slideFiles.forEach(slideFile => {
                    const fileName = path.basename(slideFile);
                    const targetFile = path.join(targetPath, fileName);
                    fs.copyFileSync(slideFile, targetFile);
                });

                slideCount = slideFiles.length;

                // Clean up temp directory
                fs.rmSync(tempExtractPath, { recursive: true, force: true });

            } catch (extractError) {
                // Clean up temp directory on error
                if (fs.existsSync(tempExtractPath)) {
                    fs.rmSync(tempExtractPath, { recursive: true, force: true });
                }
                throw extractError;
            }

            // Clean up uploaded file
            fs.unlinkSync(req.file.path);

        } else if (type === 'folder' && existingFolder) {
            // Handle existing folder
            const sourcePath = path.join(presentationsPath, existingFolder);
            if (!fs.existsSync(sourcePath)) {
                return res.status(400).json({ error: 'Source folder not found' });
            }

            // If folder names are different, copy to new location
            if (existingFolder !== folderName) {
                if (!fs.existsSync(targetPath)) {
                    fs.mkdirSync(targetPath, { recursive: true });
                }

                // Copy files
                const files = fs.readdirSync(sourcePath);
                files.forEach(file => {
                    if (file !== 'auth.json') {
                        fs.copyFileSync(
                            path.join(sourcePath, file),
                            path.join(targetPath, file)
                        );
                    }
                });
            }

            // Count slides
            const files = fs.readdirSync(targetPath);
            slideCount = files.filter(file => file.match(/^Slide_\d+\.html$/)).length;
        }

        // Apply slide updates using Node.js equivalent of update_slides.py
        await updateSlidesResponsive(targetPath);

        // Create auth.json
        const authData = {
            presentationName,
            password,
            description,
            lastUpdated: new Date().toISOString().split('T')[0]
        };

        fs.writeFileSync(
            path.join(targetPath, 'auth.json'),
            JSON.stringify(authData, null, 2)
        );

        res.json({
            success: true,
            folder: folderName,
            slides: slideCount,
            message: 'Presentation imported successfully'
        });

    } catch (error) {
        console.error('Import error:', error);
        res.status(500).json({ error: error.message });
    }
});

// Function to update slides with responsive design (Node.js version of update_slides.py)
async function updateSlidesResponsive(presentationPath) {
    try {
        const files = fs.readdirSync(presentationPath);
        const slideFiles = files.filter(file => file.match(/^Slide_\d+\.html$/));

        for (const slideFile of slideFiles) {
            const slidePath = path.join(presentationPath, slideFile);
            let content = fs.readFileSync(slidePath, 'utf-8');

            // Replace the old body and slide CSS
            const oldPattern = /\s*body\s*\{[^}]+\}\s*\.slide\s*\{[^}]+\}/gs;
            const newCSS = `        html, body {
            width: 100%;
            height: 100%;
            font-family: 'Source Sans Pro', sans-serif;
            overflow: hidden;
            margin: 0;
            padding: 0;
        }
        .slide {
            width: 100vw;
            height: 100vh;
            aspect-ratio: 16/9;
            max-width: 100vw;
            max-height: 100vh;
            position: relative;
            background: linear-gradient(135deg, rgba(25, 25, 112, 0.9), rgba(102, 51, 153, 0.9));
            color: white;
            display: flex;
            flex-direction: column;
            padding: 4vh 5vw;
            overflow: hidden;
        }

        /* Ensure 16:9 aspect ratio on all screen sizes */
        @media (max-aspect-ratio: 16/9) {
            .slide {
                width: calc(100vh * 16/9);
                height: 100vh;
                margin: 0 auto;
            }
        }

        @media (min-aspect-ratio: 16/9) {
            .slide {
                width: 100vw;
                height: calc(100vw * 9/16);
                margin: auto 0;
            }
        }`;

            content = content.replace(oldPattern, newCSS);

            // Update header styles
            content = content.replace(
                /\.header\s*\{\s*margin-bottom:\s*\d+px;\s*\}/g,
                '.header {\n            margin-bottom: clamp(2vh, 4vh, 6vh);\n        }'
            );

            // Update title styles
            content = content.replace(
                /\.title\s*\{\s*font-size:\s*\d+px;\s*font-weight:\s*700;\s*margin-bottom:\s*\d+px;\s*\}/g,
                '.title {\n            font-size: clamp(2.5rem, 4vw, 3.5rem);\n            font-weight: 700;\n            margin-bottom: clamp(0.8rem, 1.5vh, 1.2rem);\n        }'
            );

            // Add mobile breakpoints before </style>
            const mobileCSS = `
        /* Mobile responsive breakpoints */
        @media (max-width: 768px) {
            .slide {
                padding: 3vh 4vw;
            }
            .content {
                flex-direction: column;
                gap: clamp(1.5rem, 3vh, 2rem);
            }
        }

        @media (max-width: 480px) {
            .slide {
                padding: 2vh 3vw;
            }
            .title {
                font-size: clamp(1.8rem, 6vw, 2.2rem);
            }
        }`;

            content = content.replace(/(\s+)<\/style>/g, mobileCSS + '$1</style>');

            fs.writeFileSync(slidePath, content);
            console.log(`Updated ${slideFile}`);
        }
    } catch (error) {
        console.error('Error updating slides:', error);
        throw error;
    }
}

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

// API endpoint to delete a presentation
app.delete('/api/delete-presentation/:presentationKey', async (req, res) => {
    try {
        const { presentationKey } = req.params;
        const { deletePassword } = req.body;

        // Validate presentation key
        if (!presentationKey || typeof presentationKey !== 'string') {
            return res.status(400).json({
                success: false,
                error: 'Invalid presentation key'
            });
        }

        // Verify delete password (optional backend verification)
        if (deletePassword) {
            try {
                const deleteConfigPath = path.join(__dirname, 'delete-config.json');
                const deleteConfig = JSON.parse(fs.readFileSync(deleteConfigPath, 'utf-8'));

                if (deletePassword !== deleteConfig.deletePassword) {
                    return res.status(403).json({
                        success: false,
                        error: 'Invalid delete password'
                    });
                }
            } catch (configError) {
                console.error('Error reading delete config:', configError);
                return res.status(500).json({
                    success: false,
                    error: 'Unable to verify delete permissions'
                });
            }
        }

        // Sanitize the presentation key to prevent path traversal
        const sanitizedKey = presentationKey.replace(/[^a-zA-Z0-9_-]/g, '');
        if (sanitizedKey !== presentationKey) {
            return res.status(400).json({
                success: false,
                error: 'Invalid characters in presentation key'
            });
        }

        const presentationPath = path.join(__dirname, 'presentations', sanitizedKey);

        // Check if presentation exists
        if (!fs.existsSync(presentationPath)) {
            return res.status(404).json({
                success: false,
                error: 'Presentation not found'
            });
        }

        // Verify it's actually a directory
        const stat = fs.statSync(presentationPath);
        if (!stat.isDirectory()) {
            return res.status(400).json({
                success: false,
                error: 'Invalid presentation structure'
            });
        }

        // Delete the entire presentation directory
        fs.rmSync(presentationPath, { recursive: true, force: true });

        console.log(`Presentation deleted: ${sanitizedKey}`);

        res.json({
            success: true,
            message: `Presentation "${sanitizedKey}" deleted successfully`
        });

    } catch (error) {
        console.error('Delete presentation error:', error);
        res.status(500).json({
            success: false,
            error: 'Failed to delete presentation: ' + error.message
        });
    }
});

app.listen(PORT, () => {
    console.log(`🚀 KNIRV Presentation Server running at http://localhost:${PORT}`);
    console.log(`📊 Slider: http://localhost:${PORT}/presentation-slider.html`);
    console.log(`🏠 Index: http://localhost:${PORT}/index.html`);
    console.log(`📁 Serving from: ${__dirname}`);
});
