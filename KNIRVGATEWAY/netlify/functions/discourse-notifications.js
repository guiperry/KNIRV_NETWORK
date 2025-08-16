const { db, verifyToken } = require('./discourse-utils');

exports.handler = async (event, context) => {
    const { httpMethod, queryStringParameters } = event;
    const authHeader = event.headers.authorization;
    const user = verifyToken(authHeader?.replace('Bearer ', ''));

    if (!user) {
        return {
            statusCode: 401,
            body: JSON.stringify({ error: 'Authentication required' })
        };
    }

    try {
        if (httpMethod === 'GET') {
            return await getUserNotifications(user.userId, queryStringParameters);
        }

        return {
            statusCode: 405,
            body: JSON.stringify({ error: 'Method not allowed' })
        };

    } catch (error) {
        console.error('Notifications function error:', error);
        return {
            statusCode: 500,
            body: JSON.stringify({ error: 'Internal server error' })
        };
    }
};

async function getUserNotifications(userId, params) {
    const page = parseInt(params?.page) || 1;
    const limit = parseInt(params?.limit) || 20;
    const offset = (page - 1) * limit;

    const notifications = await db.findRecords('notifications', {
        user_id: userId,
        orderBy: { column: 'created_at', ascending: false },
        limit,
        offset
    });

    return {
        statusCode: 200,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ notifications })
    };
}