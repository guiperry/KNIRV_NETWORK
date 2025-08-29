import { Hono } from "hono";

type Env = Record<string, unknown>;

const app = new Hono<{ Bindings: Env }>();

export default app;
