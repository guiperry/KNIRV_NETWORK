const { db, sanitizeHtml } = require('./discourse-utils');

exports.handler = async (event, context) => {
    const { httpMethod, queryStringParameters } = event;

    try {
        if (httpMethod === 'GET') {
            return await search(queryStringParameters);
        }

        return {
            statusCode: 405,
            body: JSON.stringify({ error: 'Method not allowed' })
        };

    } catch (error) {
        console.error('Search function error:', error);
        return {
            statusCode: 500,
            body: JSON.stringify({ error: 'Internal server error' })
        };
    }
};

async function search(params) {
    const { q: query, type = 'all', page = 1, limit = 20 } = params;

    if (!query) {
        return {
            statusCode: 400,
            body: JSON.stringify({ error: 'Search query is required' })
        };
    }

    const results = {
        topics: [],
        posts: [],
        users: []
    };

    const searchTerm = `%${query.toLowerCase()}%`;

    if (type === 'all' || type === 'topics') {
        results.topics = await db.findRecords('topics', {
            $or: [
                { title: searchTerm },
                { slug: searchTerm }
            ],
            limit: parseInt(limit)
        });
    }

    if (type === 'all' || type === 'posts') {
        results.posts = await db.findRecords('posts', {
            $or: [
                { raw: searchTerm },
                { cooked: searchTerm }
            ],
            limit: parseInt(limit)
        });
    }

    if (type === 'all' || type === 'users') {
        results.users = await db.findRecords('users', {
            $or: [
                { username: searchTerm },
                { name: searchTerm }
            ],
            limit: parseInt(limit)
        });
    }

    return {
        statusCode: 200,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            query,
            results,
            total: results.topics.length + results.posts.length + results.users.length
        })
    };
}