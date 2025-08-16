const { db, verifyToken } = require('./discourse-utils');

exports.handler = async (event, context) => {
    try {
        return {
            statusCode: 200,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                message: 'ai_llm_quotas endpoint',
                data: []
            })
        };
    } catch (error) {
        console.error('ai_llm_quotas function error:', error);
        return {
            statusCode: 500,
            body: JSON.stringify({ error: 'Internal server error' })
        };
    }
};