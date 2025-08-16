const { db, verifyToken } = require('./discourse-utils');

exports.handler = async (event, context) => {
    const { httpMethod, queryStringParameters } = event;

    try {
        if (httpMethod === 'GET') {
            return await getBadges(queryStringParameters);
        }

        return {
            statusCode: 405,
            body: JSON.stringify({ error: 'Method not allowed' })
        };

    } catch (error) {
        console.error('Badges function error:', error);
        return {
            statusCode: 500,
            body: JSON.stringify({ error: 'Internal server error' })
        };
    }
};

async function getBadges(params) {
    const badges = await db.findRecords('badges', {
        enabled: true,
        orderBy: { column: 'badge_grouping_id', ascending: true }
    });

    return {
        statusCode: 200,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ badges })
    };
}