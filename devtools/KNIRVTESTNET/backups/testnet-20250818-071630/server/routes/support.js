const express = require('express');
const router = express.Router();

// Support health check
router.get('/health', (req, res) => {
  res.json({
    status: 'healthy',
    service: 'KNIRV Support',
    timestamp: new Date().toISOString()
  });
});

// Support ticket endpoints
router.get('/api/tickets', (req, res) => {
  res.json({
    tickets: [
      {
        id: 1,
        title: 'Sample Support Ticket',
        status: 'open',
        priority: 'medium',
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
      }
    ]
  });
});

router.post('/api/tickets', (req, res) => {
  const { title, description, priority = 'medium' } = req.body;
  
  if (!title || !description) {
    return res.status(400).json({
      error: 'Title and description are required'
    });
  }
  
  res.status(201).json({
    id: Date.now(),
    title,
    description,
    priority,
    status: 'open',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString()
  });
});

router.get('/api/tickets/:id', (req, res) => {
  const ticketId = req.params.id;
  
  res.json({
    id: ticketId,
    title: `Support Ticket #${ticketId}`,
    description: 'Sample ticket description',
    status: 'open',
    priority: 'medium',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    messages: [
      {
        id: 1,
        content: 'Initial support request',
        author: 'user',
        created_at: new Date().toISOString()
      }
    ]
  });
});

// Knowledge base endpoints
router.get('/api/kb/articles', (req, res) => {
  res.json({
    articles: [
      {
        id: 1,
        title: 'Getting Started with KNIRV Network',
        slug: 'getting-started',
        category: 'basics',
        created_at: new Date().toISOString()
      },
      {
        id: 2,
        title: 'Troubleshooting Common Issues',
        slug: 'troubleshooting',
        category: 'support',
        created_at: new Date().toISOString()
      }
    ]
  });
});

router.get('/api/kb/articles/:slug', (req, res) => {
  const slug = req.params.slug;
  
  res.json({
    id: 1,
    title: `Article: ${slug}`,
    slug,
    content: `# ${slug}\n\nThis is a sample knowledge base article for the KNIRV Network testnet.`,
    category: 'general',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString()
  });
});

module.exports = router;
