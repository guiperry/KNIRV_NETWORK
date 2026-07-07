const { createServer } = require('http');
const { parse } = require('url');
const next = require('next');
const http = require('http');

const dev = process.env.NODE_ENV !== 'production';
const app = next({ dev });
const handle = app.getRequestHandler();

const PORT = process.env.PORT || 3000;
const BACKEND_PORT = process.env.BACKEND_PORT || 8090;
const BACKEND_SOCKET = process.env.BACKEND_SOCKET || process.env.KNIRV_API_SOCKET || process.env.KNIRV_API_SOCKET_PATH || '';

const buildBackendRequestOptions = (req) => {
  if (BACKEND_SOCKET) {
    return {
      socketPath: BACKEND_SOCKET,
      path: req.url,
      method: req.method,
      headers: {
        ...req.headers,
        host: 'localhost',
      },
    };
  }

  return {
    hostname: '127.0.0.1',
    port: BACKEND_PORT,
    path: req.url,
    method: req.method,
    headers: {
      ...req.headers,
      host: `localhost:${BACKEND_PORT}`,
    },
  };
};

const writeUpgradeResponse = (socket, proxyRes) => {
  const statusCode = proxyRes.statusCode || 101;
  const statusMessage = proxyRes.statusMessage || 'Switching Protocols';
  const headers = Object.entries(proxyRes.headers)
    .flatMap(([key, value]) => {
      if (Array.isArray(value)) {
        return value.map((entry) => `${key}: ${entry}`);
      }
      if (typeof value === 'undefined') {
        return [];
      }
      return [`${key}: ${value}`];
    })
    .join('\r\n');

  socket.write(
    `HTTP/1.1 ${statusCode} ${statusMessage}\r\n${headers}\r\n\r\n`
  );
};

app.prepare().then(() => {
  const server = createServer(async (req, res) => {
    try {
      const parsedUrl = parse(req.url, true);
      const { pathname } = parsedUrl;

      // Proxy API requests to the backend server
      if (pathname.startsWith('/api/')) {
        const requestOptions = buildBackendRequestOptions(req);

        const proxyReq = http.request(
          requestOptions,
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
  });

  server.on('upgrade', (req, socket, head) => {
    if (!req.url || !req.url.startsWith('/ws')) {
      socket.destroy();
      return;
    }

    const proxyReq = http.request(buildBackendRequestOptions(req));

    proxyReq.on('upgrade', (proxyRes, proxySocket, proxyHead) => {
      writeUpgradeResponse(socket, proxyRes);

      if (head && head.length > 0) {
        proxySocket.write(head);
      }
      if (proxyHead && proxyHead.length > 0) {
        socket.write(proxyHead);
      }

      proxySocket.pipe(socket);
      socket.pipe(proxySocket);
    });

    proxyReq.on('error', (error) => {
      console.error('Backend WebSocket proxy error:', error);
      socket.destroy();
    });

    proxyReq.end();
  });

  server.listen(PORT, (err) => {
    if (err) throw err;
    console.log(`> KNIRVSERVER Frontend Server ready on http://localhost:${PORT}`);
    if (BACKEND_SOCKET) {
      console.log(`> Proxying API and WebSocket requests to backend socket ${BACKEND_SOCKET}`);
    } else {
      console.log(`> Proxying API and WebSocket requests to backend on http://localhost:${BACKEND_PORT}`);
    }
  });
});
