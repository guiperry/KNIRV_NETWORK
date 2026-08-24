import type { NextApiRequest, NextApiResponse } from "next";

// Keep browser traffic same-origin. KNIRVSERVER remains the only integration
// point; this route deliberately contains no business or signing logic.
const backend = process.env.KNIRVSERVER_BACKEND_URL || "http://localhost:8082";

export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  const path = req.query.path;
  if (!Array.isArray(path) || path.some(part => !/^[A-Za-z0-9._-]+$/.test(part))) {
    return res.status(400).json({ error: "invalid actuarial path" });
  }
  const target = `${backend}/api/v1/actuarial/${path.map(encodeURIComponent).join("/")}${req.url?.includes("?") ? req.url.slice(req.url.indexOf("?")) : ""}`;
  const headers: Record<string, string> = {};
  for (const name of ["authorization", "content-type", "idempotency-key", "x-knirv-network-id", "x-knirv-wallet"]) {
    const value = req.headers[name];
    if (typeof value === "string") headers[name] = value;
  }
  const upstream = await fetch(target, {
    method: req.method,
    headers,
    body: ["GET", "HEAD"].includes(req.method ?? "GET") ? undefined : JSON.stringify(req.body ?? {}),
  });
  const body = await upstream.text();
  res.status(upstream.status);
  const contentType = upstream.headers.get("content-type");
  if (contentType) res.setHeader("content-type", contentType);
  res.send(body);
}
