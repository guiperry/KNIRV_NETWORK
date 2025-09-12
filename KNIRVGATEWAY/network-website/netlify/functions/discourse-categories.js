const { db, verifyToken, sanitizeHtml } = require('./discourse-utils');

exports.handler = async (event, context) => {
    const { httpMethod, queryStringParameters, body } = event;

    try {
        if (httpMethod === 'GET') {
            return await getCategories(queryStringParameters);
        } else if (httpMethod === 'POST') {
            const authHeader = event.headers.authorization;
            const user = verifyToken(authHeader?.replace('Bearer ', ''));

            if (!user || (!user.admin && !user.moderator)) {
                return {
                    statusCode: 403,
                    body: JSON.stringify({ error: 'Admin access required' })
                };
            }

            return await createCategory(JSON.parse(body), user);
        }

        return {
            statusCode: 405,
            body: JSON.stringify({ error: 'Method not allowed' })
        };

    } catch (error) {
        console.error('Categories function error:', error);
        return {
            statusCode: 500,
            body: JSON.stringify({ error: 'Internal server error' })
        };
    }
};

async function getCategories(params) {
    const categories = await db.findRecords('categories', {
        orderBy: { column: 'position', ascending: true }
    });

    // Get topic counts for each category
    for (const category of categories) {
        const topics = await db.findRecords('topics', { category_id: category.id });
        category.topic_count = topics.length;

        // Get subcategories
        const subcategories = await db.findRecords('categories', {
            parent_category_id: category.id
        });
        category.subcategories = subcategories;
    }

    return {
        statusCode: 200,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ categories })
    };
}