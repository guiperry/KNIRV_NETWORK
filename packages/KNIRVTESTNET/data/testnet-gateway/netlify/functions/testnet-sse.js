// Testnet SSE Endpoint
const handler = async (event, context) => {
  // Set SSE headers
  const headers = {
    'Content-Type': 'text/event-stream',
    'Cache-Control': 'no-cache',
    'Connection': 'keep-alive',
    'Access-Control-Allow-Origin': '*'
  };

  return {
    statusCode: 200,
    headers,
    body: 'SSE connection established for testnet'
  };
};

exports.handler = handler;