const { db, verifyToken } = require('./discourse-utils');

exports.handler = async (event, context) => {
    try {
        return {
            statusCode: 200,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                message: 'theme_javascripts endpoint',
                data: []
            })
        };
    } catch (error) {
        console.error('theme_javascripts function error:', error);
        return {
            statusCode: 500,
            body: JSON.stringify({ error: 'Internal server error' })
        };
    }
};