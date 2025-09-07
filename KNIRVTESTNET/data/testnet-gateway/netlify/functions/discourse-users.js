const { db, verifyToken, sanitizeHtml } = require('./discourse-utils');
const bcrypt = require('bcryptjs');

exports.handler = async (event, context) => {
    const { httpMethod, path: requestPath, queryStringParameters, body } = event;

    try {
        if (httpMethod === 'GET') {
            if (requestPath.includes('/u/')) {
                const username = requestPath.split('/u/')[1];
                return await getUserProfile(username);
            } else {
                return await getUsersList(queryStringParameters);
            }
        } else if (httpMethod === 'POST') {
            return await createUser(JSON.parse(body));
        } else if (httpMethod === 'PUT') {
            const authHeader = event.headers.authorization;
            const user = verifyToken(authHeader?.replace('Bearer ', ''));

            if (!user) {
                return {
                    statusCode: 401,
                    body: JSON.stringify({ error: 'Authentication required' })
                };
            }

            return await updateUser(JSON.parse(body), user);
        }

        return {
            statusCode: 405,
            body: JSON.stringify({ error: 'Method not allowed' })
        };

    } catch (error) {
        console.error('Users function error:', error);
        return {
            statusCode: 500,
            body: JSON.stringify({ error: 'Internal server error' })
        };
    }
};

async function getUserProfile(username) {
    const users = await db.findRecords('users', { username });

    if (users.length === 0) {
        return {
            statusCode: 404,
            body: JSON.stringify({ error: 'User not found' })
        };
    }

    const user = users[0];

    // Get user's recent topics and posts
    const topics = await db.findRecords('topics', {
        user_id: user.id,
        orderBy: { column: 'created_at', ascending: false },
        limit: 10
    });

    const posts = await db.findRecords('posts', {
        user_id: user.id,
        orderBy: { column: 'created_at', ascending: false },
        limit: 20
    });

    // Get user's badges
    const userBadges = await db.findRecords('user_badges', { user_id: user.id });
    const badges = [];
    for (const userBadge of userBadges) {
        const badgeRecords = await db.findRecords('badges', { id: userBadge.badge_id });
        if (badgeRecords.length > 0) {
            badges.push(badgeRecords[0]);
        }
    }

    return {
        statusCode: 200,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            user: {
                id: user.id,
                username: user.username,
                name: user.name,
                avatar_url: user.avatar_url,
                bio: user.bio,
                location: user.location,
                website: user.website,
                trust_level: user.trust_level,
                created_at: user.created_at,
                last_seen_at: user.last_seen_at
            },
            topics,
            posts,
            badges
        })
    };
}

async function createUser(data) {
    const { username, email, password, name } = data;

    if (!username || !email || !password) {
        return {
            statusCode: 400,
            body: JSON.stringify({ error: 'Username, email, and password are required' })
        };
    }

    // Check if user already exists
    const existingUsers = await db.findRecords('users', {
        $or: [{ username }, { email }]
    });

    if (existingUsers.length > 0) {
        return {
            statusCode: 409,
            body: JSON.stringify({ error: 'User already exists' })
        };
    }

    // Hash password
    const passwordHash = await bcrypt.hash(password, 10);

    // Create user
    const userData = {
        username: sanitizeHtml(username),
        email,
        password_hash: passwordHash,
        name: name ? sanitizeHtml(name) : username,
        trust_level: 0,
        admin: false,
        moderator: false,
        suspended: false,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        last_seen_at: new Date().toISOString()
    };

    const user = await db.insertRecord('users', userData);

    return {
        statusCode: 201,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            user: {
                id: user.id,
                username: user.username,
                email: user.email,
                name: user.name,
                trust_level: user.trust_level
            }
        })
    };
}