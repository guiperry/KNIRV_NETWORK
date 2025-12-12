const { db, verifyToken } = require('./discourse-utils');

exports.handler = async (event, context) => {
    const { httpMethod, queryStringParameters } = event;

    try {
        if (httpMethod === 'GET') {
            return await getTags(queryStringParameters);
        }

        return {
            statusCode: 405,
            body: JSON.stringify({ error: 'Method not allowed' })
        };

    } catch (error) {
        console.error('Tags function error:', error);
        return {
            statusCode: 500,
            body: JSON.stringify({ error: 'Internal server error' })
        };
    }
};

async function getTags(params) {
    const tags = await db.findRecords('tags', {
        orderBy: { column: 'topic_count', ascending: false },
        limit: 50
    });

    return {
        statusCode: 200,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tags })
    };
}