const { db, verifyToken } = require('./discourse-utils');

exports.handler = async (event, context) => {
    try {
        return {
            statusCode: 200,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                message: 'channel_threads_current_user_title_prompt_seen endpoint',
                data: []
            })
        };
    } catch (error) {
        console.error('channel_threads_current_user_title_prompt_seen function error:', error);
        return {
            statusCode: 500,
            body: JSON.stringify({ error: 'Internal server error' })
        };
    }
};