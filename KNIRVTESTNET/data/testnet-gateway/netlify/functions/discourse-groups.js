const { db, verifyToken } = require('./discourse-utils');

exports.handler = async (event, context) => {
    const { httpMethod, queryStringParameters } = event;

    try {
        if (httpMethod === 'GET') {
            return await getGroups(queryStringParameters);
        }

        return {
            statusCode: 405,
            body: JSON.stringify({ error: 'Method not allowed' })
        };

    } catch (error) {
        console.error('Groups function error:', error);
        return {
            statusCode: 500,
            body: JSON.stringify({ error: 'Internal server error' })
        };
    }
};

async function getGroups(params) {
    const groups = await db.findRecords('groups', {
        orderBy: { column: 'name', ascending: true }
    });

    return {
        statusCode: 200,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ groups })
    };
}