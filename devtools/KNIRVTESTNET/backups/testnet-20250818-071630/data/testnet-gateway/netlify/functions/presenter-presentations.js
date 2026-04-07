const fs = require('fs');
const path = require('path');

exports.handler = async (event, context) => {
    // Set CORS headers
    const headers = {
        'Access-Control-Allow-Origin': '*',
        'Access-Control-Allow-Headers': 'Content-Type',
        'Access-Control-Allow-Methods': 'GET, POST, OPTIONS',
    };

    // Handle preflight requests
    if (event.httpMethod === 'OPTIONS') {
        return {
            statusCode: 200,
            headers,
            body: '',
        };
    }

    if (event.httpMethod !== 'GET') {
        return {
            statusCode: 405,
            headers,
            body: JSON.stringify({ error: 'Method not allowed' }),
        };
    }

    try {
        // In Netlify Functions, we need to construct the path to the presenter directory
        // The function runs in a different context, so we need to find the presentations folder
        const presentationsPath = path.join(process.cwd(), 'presenter', 'presentations');
        const presentations = {};

        if (fs.existsSync(presentationsPath)) {
            const dirs = fs.readdirSync(presentationsPath).filter(item => {
                const itemPath = path.join(presentationsPath, item);
                return fs.statSync(itemPath).isDirectory();
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

        return {
            statusCode: 200,
            headers,
            body: JSON.stringify({ presentations }),
        };

    } catch (error) {
        console.error('Error loading presentations:', error);
        return {
            statusCode: 500,
            headers,
            body: JSON.stringify({ 
                error: 'Failed to load presentations',
                message: error.message 
            }),
        };
    }
};
