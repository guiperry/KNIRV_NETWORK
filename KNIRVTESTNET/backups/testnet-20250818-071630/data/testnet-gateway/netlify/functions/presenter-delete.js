const fs = require('fs');
const path = require('path');

exports.handler = async (event, context) => {
    // Set CORS headers
    const headers = {
        'Access-Control-Allow-Origin': '*',
        'Access-Control-Allow-Headers': 'Content-Type',
        'Access-Control-Allow-Methods': 'GET, POST, DELETE, OPTIONS',
    };

    // Handle preflight requests
    if (event.httpMethod === 'OPTIONS') {
        return {
            statusCode: 200,
            headers,
            body: '',
        };
    }

    if (event.httpMethod !== 'DELETE') {
        return {
            statusCode: 405,
            headers,
            body: JSON.stringify({ error: 'Method not allowed' }),
        };
    }

    try {
        // Extract presentation key from path
        const pathParts = event.path.split('/');
        const presentationKey = pathParts[pathParts.length - 1];
        
        if (!presentationKey) {
            return {
                statusCode: 400,
                headers,
                body: JSON.stringify({ 
                    success: false, 
                    error: 'Presentation key is required' 
                }),
            };
        }

        // For Netlify deployment, deletion is not available
        // This would require write access to the repository
        return {
            statusCode: 200,
            headers,
            body: JSON.stringify({
                success: false,
                error: 'Deletion is not available in the deployed version.',
                message: 'To delete presentations, please remove them manually from the repository and redeploy.',
                suggestion: 'For local development, use the standalone presenter server for full delete functionality.'
            }),
        };

    } catch (error) {
        console.error('Delete error:', error);
        return {
            statusCode: 500,
            headers,
            body: JSON.stringify({
                success: false,
                error: 'Internal server error',
                message: error.message
            }),
        };
    }
};
