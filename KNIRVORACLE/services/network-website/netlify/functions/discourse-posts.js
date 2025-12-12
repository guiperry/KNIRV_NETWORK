const { db, verifyToken, sanitizeHtml } = require('./discourse-utils');

exports.handler = async (event, context) => {
    const { httpMethod, queryStringParameters, body } = event;

    try {
        if (httpMethod === 'POST') {
            const authHeader = event.headers.authorization;
            const user = verifyToken(authHeader?.replace('Bearer ', ''));

            if (!user) {
                return {
                    statusCode: 401,
                    body: JSON.stringify({ error: 'Authentication required' })
                };
            }

            return await createPost(JSON.parse(body), user);
        } else if (httpMethod === 'PUT') {
            const authHeader = event.headers.authorization;
            const user = verifyToken(authHeader?.replace('Bearer ', ''));

            if (!user) {
                return {
                    statusCode: 401,
                    body: JSON.stringify({ error: 'Authentication required' })
                };
            }

            const postId = queryStringParameters?.id;
            return await updatePost(postId, JSON.parse(body), user);
        } else if (httpMethod === 'DELETE') {
            const authHeader = event.headers.authorization;
            const user = verifyToken(authHeader?.replace('Bearer ', ''));

            if (!user) {
                return {
                    statusCode: 401,
                    body: JSON.stringify({ error: 'Authentication required' })
                };
            }

            const postId = queryStringParameters?.id;
            return await deletePost(postId, user);
        }

        return {
            statusCode: 405,
            body: JSON.stringify({ error: 'Method not allowed' })
        };

    } catch (error) {
        console.error('Posts function error:', error);
        return {
            statusCode: 500,
            body: JSON.stringify({ error: 'Internal server error' })
        };
    }
};

async function createPost(data, user) {
    const { topic_id, raw, reply_to_post_number } = data;

    if (!topic_id || !raw) {
        return {
            statusCode: 400,
            body: JSON.stringify({ error: 'Topic ID and content are required' })
        };
    }

    // Get topic to determine post number
    const topics = await db.findRecords('topics', { id: parseInt(topic_id) });
    if (topics.length === 0) {
        return {
            statusCode: 404,
            body: JSON.stringify({ error: 'Topic not found' })
        };
    }

    const topic = topics[0];
    const postNumber = (topic.posts_count || 0) + 1;

    // Create post
    const postData = {
        topic_id: parseInt(topic_id),
        user_id: user.userId,
        post_number: postNumber,
        raw: sanitizeHtml(raw),
        cooked: sanitizeHtml(raw),
        reply_to_post_number: reply_to_post_number || null,
        like_count: 0,
        reply_count: 0,
        quote_count: 0,
        hidden: false,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
    };

    const post = await db.insertRecord('posts', postData);

    // Update topic stats
    await db.updateRecord('topics', parseInt(topic_id), {
        posts_count: postNumber,
        reply_count: postNumber - 1,
        last_posted_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
    });

    return {
        statusCode: 201,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ post })
    };
}