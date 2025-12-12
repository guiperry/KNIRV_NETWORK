const { db, verifyToken } = require('./discourse-utils');
const formidable = require('formidable');
const fs = require('fs');
const path = require('path');

exports.handler = async (event, context) => {
    const { httpMethod } = event;
    const authHeader = event.headers.authorization;
    const user = verifyToken(authHeader?.replace('Bearer ', ''));

    if (!user) {
        return {
            statusCode: 401,
            body: JSON.stringify({ error: 'Authentication required' })
        };
    }

    try {
        if (httpMethod === 'POST') {
            return await handleUpload(event, user);
        }

        return {
            statusCode: 405,
            body: JSON.stringify({ error: 'Method not allowed' })
        };

    } catch (error) {
        console.error('Upload function error:', error);
        return {
            statusCode: 500,
            body: JSON.stringify({ error: 'Internal server error' })
        };
    }
};

async function handleUpload(event, user) {
    const form = formidable({
        multiples: true,
        maxFileSize: 10 * 1024 * 1024,
        uploadDir: '/tmp'
    });

    const [fields, files] = await form.parse(event.body);

    const uploadedFiles = [];
    const fileArray = Array.isArray(files.file) ? files.file : [files.file];

    for (const file of fileArray) {
        if (file && file.size > 0) {
            // In a real implementation, upload to cloud storage
            const fileRecord = {
                original_name: file.originalFilename,
                file_size: file.size,
                mime_type: file.mimetype,
                user_id: user.userId,
                created_at: new Date().toISOString()
            };

            const savedFile = await db.insertRecord('uploads', fileRecord);
            uploadedFiles.push(savedFile);
        }
    }

    return {
        statusCode: 200,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ uploads: uploadedFiles })
    };
}