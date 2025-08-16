const { db, verifyToken, sanitizeHtml } = require('./discourse-utils');
const config = require('./config-loader');

exports.handler = async (event, context) => {
    const { httpMethod, path: requestPath, queryStringParameters, body } = event;

    try {
        if (httpMethod === 'GET') {
            if (requestPath.includes('/latest')) {
                return await getLatestTopics(queryStringParameters);
            } else if (requestPath.includes('/top')) {
                return await getTopTopics(queryStringParameters);
            } else if (requestPath.includes('/new')) {
                return await getNewTopics(queryStringParameters);
            } else if (requestPath.match(/\/t\/\d+/)) {
                const topicId = requestPath.match(/\/t\/(\d+)/)[1];
                return await getTopic(topicId, queryStringParameters);
            } else {
                return await getTopicsList(queryStringParameters);
            }
        } else if (httpMethod === 'POST') {
            const authHeader = event.headers.authorization;
            const user = verifyToken(authHeader?.replace('Bearer ', ''));

            if (!user) {
                return {
                    statusCode: 401,
                    body: JSON.stringify({ error: 'Authentication required' })
                };
            }

            return await createTopic(JSON.parse(body), user);
        } else if (httpMethod === 'PUT') {
            const authHeader = event.headers.authorization;
            const user = verifyToken(authHeader?.replace('Bearer ', ''));

            if (!user) {
                return {
                    statusCode: 401,
                    body: JSON.stringify({ error: 'Authentication required' })
                };
            }

            const topicId = requestPath.match(/\/t\/(\d+)/)[1];
            return await updateTopic(topicId, JSON.parse(body), user);
        }

        return {
            statusCode: 405,
            body: JSON.stringify({ error: 'Method not allowed' })
        };

    } catch (error) {
        console.error('Topics function error:', error);
        return {
            statusCode: 500,
            body: JSON.stringify({ error: 'Internal server error' })
        };
    }
};

async function getLatestTopics(params) {
    const page = parseInt(params?.page) || 1;
    const limit = parseInt(params?.limit) || 30;
    const offset = (page - 1) * limit;

    const topics = await db.findRecords('topics', {
        orderBy: { column: 'last_posted_at', ascending: false },
        limit,
        offset
    });

    // Get category and user info for each topic
    for (const topic of topics) {
        if (topic.category_id) {
            const categories = await db.findRecords('categories', { id: topic.category_id });
            topic.category = categories[0];
        }

        if (topic.user_id) {
            const users = await db.findRecords('users', { id: topic.user_id });
            topic.user = users[0] ? {
                id: users[0].id,
                username: users[0].username,
                avatar_url: users[0].avatar_url
            } : null;
        }
    }

    return {
        statusCode: 200,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            topics,
            pagination: {
                page,
                limit,
                hasMore: topics.length === limit
            }
        })
    };
}

async function getTopTopics(params) {
    const period = params?.period || 'all';
    const page = parseInt(params?.page) || 1;
    const limit = parseInt(params?.limit) || 30;
    const offset = (page - 1) * limit;

    let dateFilter = {};
    if (period === 'week') {
        const weekAgo = new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString();
        dateFilter = { created_at: { $gte: weekAgo } };
    } else if (period === 'month') {
        const monthAgo = new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString();
        dateFilter = { created_at: { $gte: monthAgo } };
    }

    const topics = await db.findRecords('topics', {
        ...dateFilter,
        orderBy: { column: 'like_count', ascending: false },
        limit,
        offset
    });

    return {
        statusCode: 200,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ topics })
    };
}

async function getTopic(topicId, params) {
    const topics = await db.findRecords('topics', { id: parseInt(topicId) });

    if (topics.length === 0) {
        return {
            statusCode: 404,
            body: JSON.stringify({ error: 'Topic not found' })
        };
    }

    const topic = topics[0];

    // Get posts for this topic
    const posts = await db.findRecords('posts', {
        topic_id: parseInt(topicId),
        orderBy: { column: 'post_number', ascending: true }
    });

    // Get user info for each post
    for (const post of posts) {
        if (post.user_id) {
            const users = await db.findRecords('users', { id: post.user_id });
            post.user = users[0] ? {
                id: users[0].id,
                username: users[0].username,
                avatar_url: users[0].avatar_url,
                trust_level: users[0].trust_level
            } : null;
        }
    }

    // Increment view count
    await db.updateRecord('topics', parseInt(topicId), {
        views: (topic.views || 0) + 1
    });

    return {
        statusCode: 200,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            topic,
            posts
        })
    };
}

async function createTopic(data, user) {
    const { title, raw, category_id, tags } = data;

    if (!title || !raw) {
        return {
            statusCode: 400,
            body: JSON.stringify({ error: 'Title and content are required' })
        };
    }

    // Create topic
    const topicData = {
        title: sanitizeHtml(title),
        slug: title.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, ''),
        category_id: category_id || null,
        user_id: user.userId,
        views: 0,
        posts_count: 1,
        reply_count: 0,
        like_count: 0,
        pinned: false,
        closed: false,
        archived: false,
        visible: true,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        last_posted_at: new Date().toISOString()
    };

    const topic = await db.insertRecord('topics', topicData);

    // Create first post
    const postData = {
        topic_id: topic.id,
        user_id: user.userId,
        post_number: 1,
        raw: sanitizeHtml(raw),
        cooked: sanitizeHtml(raw), // In real implementation, process markdown
        like_count: 0,
        reply_count: 0,
        quote_count: 0,
        hidden: false,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
    };

    const post = await db.insertRecord('posts', postData);

    // Handle tags if provided
    if (tags && Array.isArray(tags)) {
        for (const tagName of tags) {
            let tag = await db.findRecords('tags', { name: tagName });
            if (tag.length === 0) {
                tag = await db.insertRecord('tags', {
                    name: tagName,
                    topic_count: 1,
                    created_at: new Date().toISOString(),
                    updated_at: new Date().toISOString()
                });
            } else {
                await db.updateRecord('tags', tag[0].id, {
                    topic_count: (tag[0].topic_count || 0) + 1
                });
                tag = tag[0];
            }

            await db.insertRecord('topic_tags', {
                topic_id: topic.id,
                tag_id: tag.id,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString()
            });
        }
    }

    return {
        statusCode: 201,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            topic,
            post
        })
    };
}