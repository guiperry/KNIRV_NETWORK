// server.ts - Next.js Custom Server + Socket.IO
// eslint-disable-next-line @typescript-eslint/no-require-imports
const { createServer } = require('http');
// eslint-disable-next-line @typescript-eslint/no-require-imports
const { Server } = require('socket.io');
// eslint-disable-next-line @typescript-eslint/no-require-imports
const next = require('next');

const dev = process.env.NODE_ENV !== 'production';
const hostname = '0.0.0.0';
const port = parseInt(process.env.PORT || '3000', 10);

// Custom server with Socket.IO integration
async function createCustomServer() {
  try {
    // Create Next.js app
    const app = next({ dev, hostname, port });
    const handle = app.getRequestHandler();

    await app.prepare();

    // Create HTTP server
    const server = createServer(async (req, res) => {
      try {
        // Skip socket.io requests from Next.js handler
        if (req.url?.startsWith('/socket.io/')) {
          return;
        }

        await handle(req, res);
      } catch (err) {
        console.error('Error occurred handling', req.url, err);
        res.statusCode = 500;
        res.end('internal server error');
      }
    });

    // Setup Socket.IO
    const io = new Server(server, {
      path: '/socket.io',
      cors: {
        origin: "*",
        methods: ["GET", "POST"]
      }
    });

    // Import and setup socket handlers
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const { setupSocket } = require('./dist/socket.cjs');
    setupSocket(io);

    // Start the server
    server.listen(port, hostname, () => {
      console.log(`> Ready on http://${hostname}:${port}`);
      console.log(`> Socket.IO server running at ws://${hostname}:${port}/socket.io`);
    });

  } catch (err) {
    console.error('Server startup error:', err);
    process.exit(1);
  }
}

// Start the server
createCustomServer();
