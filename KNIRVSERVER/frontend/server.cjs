const { createServer } = require('http');
const { parse } = require('url');
const next = require('next');

const dev = process.env.NODE_ENV !== 'production';
const app = next({ dev });
const handle = app.getRequestHandler();

const PORT = process.env.PORT || 3000;
const BACKEND_PORT = process.env.BACKEND_PORT || 8082;

app.prepare().then(() => {
  createServer(async (req, res) => {
    try {
      const parsedUrl = parse(req.url, true);
      const { pathname, query } = parsedUrl;

      // Proxy API requests to the backend server
      if (pathname.startsWith('/api/')) {
        const backendUrl = `http://localhost:${BACKEND_PORT}${pathname}`;
        
        const proxyReq = require('http').request(
          backendUrl,
          {
            method: req.method,
            headers: {
              ...req.headers,
              host: `localhost:${BACKEND_PORT}`,
            },
          },
          (proxyRes) => {
            res.writeHead(proxyRes.statusCode, proxyRes.headers);
            proxyRes.pipe(res);
          }
        );

        proxyReq.on('error', (error) => {
          console.error('Backend proxy error:', error);
          res.writeHead(503, { 'Content-Type': 'application/json' });
          res.end(JSON.stringify({ error: 'Backend service unavailable' }));
        });

        req.pipe(proxyReq);
      } else {
        // Handle Next.js routing
        await handle(req, res, parsedUrl);
      }
    } catch (err) {
      console.error('Error handling request:', err);
      res.writeHead(500, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ error: 'Internal server error' }));
    }
  }).listen(PORT, (err) => {
    if (err) throw err;
    console.log(`> KNIRVNEXUS Frontend Server ready on http://localhost:${PORT}`);
    console.log(`> Proxying API requests to backend on http://localhost:${BACKEND_PORT}`);
  });
});
