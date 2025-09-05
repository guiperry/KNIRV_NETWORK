const { db, verifyToken } = require('./discourse-utils');

exports.handler = async (event, context) => {
    const { httpMethod, path: requestPath, queryStringParameters, body } = event;
    const authHeader = event.headers.authorization;
    const user = verifyToken(authHeader?.replace('Bearer ', ''));

    if (!user || !user.admin) {
        return {
            statusCode: 403,
            body: JSON.stringify({ error: 'Admin access required' })
        };
    }

    try {
        if (httpMethod === 'GET') {
            if (requestPath.includes('/stats')) {
                return await getAdminStats();
            } else if (requestPath.includes('/users')) {
                return await getAdminUsers(queryStringParameters);
            } else if (requestPath.includes('/reports')) {
                return await getAdminReports(queryStringParameters);
            }
        }

        return {
            statusCode: 405,
            body: JSON.stringify({ error: 'Method not allowed' })
        };

    } catch (error) {
        console.error('Admin function error:', error);
        return {
            statusCode: 500,
            body: JSON.stringify({ error: 'Internal server error' })
        };
    }
};

async function getAdminStats() {
    const [users, topics, posts, categories] = await Promise.all([
        db.findRecords('users'),
        db.findRecords('topics'),
        db.findRecords('posts'),
        db.findRecords('categories')
    ]);

    const stats = {
        total_users: users.length,
        total_topics: topics.length,
        total_posts: posts.length,
        total_categories: categories.length,
        active_users: users.filter(u => {
            const lastSeen = new Date(u.last_seen_at);
            const weekAgo = new Date(Date.now() - 7 * 24 * 60 * 60 * 1000);
            return lastSeen > weekAgo;
        }).length
    };

    return {
        statusCode: 200,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ stats })
    };
}