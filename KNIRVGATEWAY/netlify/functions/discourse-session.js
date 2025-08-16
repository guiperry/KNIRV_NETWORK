const { db, verifyToken } = require('./discourse-utils');
const bcrypt = require('bcryptjs');

exports.handler = async (event, context) => {
    const { httpMethod, body } = event;

    try {
        if (httpMethod === 'POST') {
            return await login(JSON.parse(body));
        } else if (httpMethod === 'DELETE') {
            return await logout();
        }

        return {
            statusCode: 405,
            body: JSON.stringify({ error: 'Method not allowed' })
        };

    } catch (error) {
        console.error('Session function error:', error);
        return {
            statusCode: 500,
            body: JSON.stringify({ error: 'Internal server error' })
        };
    }
};

async function login(data) {
    const { login, password } = data;

    if (!login || !password) {
        return {
            statusCode: 400,
            body: JSON.stringify({ error: 'Login and password are required' })
        };
    }

    // Find user by username or email
    const users = await db.findRecords('users', {
        $or: [{ username: login }, { email: login }]
    });

    if (users.length === 0) {
        return {
            statusCode: 401,
            body: JSON.stringify({ error: 'Invalid credentials' })
        };
    }

    const user = users[0];

    // Verify password
    const passwordValid = await bcrypt.compare(password, user.password_hash);

    if (!passwordValid) {
        return {
            statusCode: 401,
            body: JSON.stringify({ error: 'Invalid credentials' })
        };
    }

    // Generate token
    const token = Buffer.from(JSON.stringify({
        userId: user.id,
        username: user.username,
        email: user.email,
        trustLevel: user.trust_level,
        admin: user.admin,
        moderator: user.moderator,
        exp: Date.now() + 24 * 60 * 60 * 1000 // 24 hours
    })).toString('base64');

    // Update last seen
    await db.updateRecord('users', user.id, {
        last_seen_at: new Date().toISOString()
    });

    return {
        statusCode: 200,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            success: true,
            token,
            user: {
                id: user.id,
                username: user.username,
                name: user.name,
                avatar_url: user.avatar_url,
                trust_level: user.trust_level,
                admin: user.admin,
                moderator: user.moderator
            }
        })
    };
}

async function logout() {
    return {
        statusCode: 200,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ success: true })
    };
}