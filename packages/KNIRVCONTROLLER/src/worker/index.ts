import { Hono, Context } from "hono";

interface Env {
  KNIRVSERVER_URL?: string;
}

const app = new Hono<{ Bindings: Env }>();

// Helper: proxy a request to the KNIRVSERVER backend
async function proxyRequest(
  c: Context<{ Bindings: Env }>,
  backendPath: string
): Promise<Response> {
  const baseUrl = c.env.KNIRVSERVER_URL;
  if (!baseUrl) {
    return new Response(
      JSON.stringify({ error: "KNIRVSERVER_URL not configured" }),
      { status: 503, headers: { "Content-Type": "application/json" } }
    );
  }

  const url = new URL(backendPath, baseUrl);
  const headers = new Headers(c.req.raw.headers);
  // Remove hop-by-hop headers that shouldn't be forwarded
  headers.delete("host");

  try {
    const response = await fetch(url.toString(), {
      method: c.req.method,
      headers,
      body: c.req.method === "GET" || c.req.method === "HEAD" ? undefined : c.req.raw.body,
    });

    // Build response with CORS headers
    const corsHeaders = new Headers(response.headers);
    corsHeaders.set("Access-Control-Allow-Origin", "*");

    return new Response(response.body, {
      status: response.status,
      statusText: response.statusText,
      headers: corsHeaders,
    });
  } catch (err) {
    console.error("Proxy error:", err);
    return new Response(
      JSON.stringify({ error: "Internal server error" }),
      { status: 500, headers: { "Content-Type": "application/json" } }
    );
  }
}

// DVE routes
app.get("/worker/dve/list", (c) => proxyRequest(c, "/api/dve/nodes"));
app.get("/worker/dve/creations", (c) => proxyRequest(c, "/api/dve/creations"));
app.post("/worker/dve/create", (c) => proxyRequest(c, "/api/dve/creations"));

// Cognitive routes
app.get("/worker/cognitive/status", (c) => proxyRequest(c, "/api/health"));
app.post("/worker/cognitive/chat", (c) => proxyRequest(c, "/api/knirvshell/execute"));

// CORS preflight handler for all /worker/* routes
app.options("/worker/:path*", (_c) => {
  return new Response(null, {
    status: 204,
    headers: {
      "Access-Control-Allow-Origin": "*",
      "Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, PATCH, OPTIONS",
      "Access-Control-Allow-Headers": "Content-Type, Authorization, X-Requested-With",
      "Access-Control-Max-Age": "86400",
    },
  });
});

export default app;
