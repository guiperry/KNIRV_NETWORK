const express = require('express');
const router = express.Router();

// Forum health check
router.get('/health', (req, res) => {
  res.json({
    status: 'healthy',
    service: 'KNIRV Forum',
    timestamp: new Date().toISOString()
  });
});

// Forum API endpoints (simplified versions of Discourse functions)
router.get('/api/categories', (req, res) => {
  res.json({
    categories: [
      {
        id: 1,
        name: 'General Discussion',
        description: 'General KNIRV Network discussion',
        slug: 'general'
      },
      {
        id: 2,
        name: 'Development',
        description: 'Development and technical discussions',
        slug: 'development'
      },
      {
        id: 3,
        name: 'Support',
        description: 'Help and support',
        slug: 'support'
      }
    ]
  });
});

router.get('/api/topics', (req, res) => {
  res.json({
    topics: [
      {
        id: 1,
        title: 'Welcome to KNIRV Network',
        category_id: 1,
        created_at: new Date().toISOString(),
        posts_count: 1
      }
    ]
  });
});

router.post('/api/topics', (req, res) => {
  const { title, category_id, raw } = req.body;
  
  if (!title || !raw) {
    return res.status(400).json({
      error: 'Title and content are required'
    });
  }
  
  res.status(201).json({
    id: Date.now(),
    title,
    category_id: category_id || 1,
    created_at: new Date().toISOString(),
    posts_count: 1
  });
});

// Forum authentication (simplified)
router.post('/api/session', (req, res) => {
  const { username, password } = req.body;
  
  // Simple mock authentication for testnet
  if (username && password) {
    res.json({
      user: {
        id: 1,
        username,
        email: `${username}@testnet.knirv.com`
      },
      token: 'mock-jwt-token-for-testnet'
    });
  } else {
    res.status(401).json({
      error: 'Invalid credentials'
    });
  }
});

module.exports = router;
