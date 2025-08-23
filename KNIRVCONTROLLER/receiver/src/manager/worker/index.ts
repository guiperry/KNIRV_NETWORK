import { Hono } from "hono";

interface Env {
  // Add environment variables here as needed
}

const app = new Hono<{ Bindings: Env }>();

export default app;
