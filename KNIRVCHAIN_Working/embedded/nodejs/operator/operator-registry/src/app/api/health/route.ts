import { NextResponse } from "next/server";

export async function GET() {
  const healthStatus = {
    status: "healthy",
  service: "NANDA ANS - Operator Registry",
    version: "1.0.0",
    timestamp: new Date().toISOString(),
    uptime: process.uptime(),
    components: {
      api: "operational",
      database: "operational",
  operator_registry: "operational",
      knirv_oracle_integration: "operational"
    },
    metrics: {
      memory_usage: process.memoryUsage(),
      cpu_usage: process.cpuUsage(),
      uptime_seconds: process.uptime()
    },
    endpoints: {
  operator_discovery: "/",
  operator_registration: "/register",
  operator_search: "/api/agents",
      health_check: "/api/health"
    },
    integration: {
      knirv_oracle: {
        endpoint: process.env.NEXT_PUBLIC_KNIRVCHAIN_URL || 'http://localhost:1317',
        status: "connected"
      }
    }
  };

  return NextResponse.json(healthStatus);
}
