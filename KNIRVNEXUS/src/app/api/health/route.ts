import { NextResponse } from "next/server";

export async function GET() {
  const healthStatus = {
    status: "healthy",
    service: "KNIRV-NEXUS DVE",
    version: "2.0.0",
    timestamp: new Date().toISOString(),
    uptime: process.uptime(),
    components: {
      api: "operational",
      websocket: "operational",
      database: "operational",
      cache: "operational"
    },
    metrics: {
      memory_usage: process.memoryUsage(),
      cpu_usage: process.cpuUsage(),
      uptime_seconds: process.uptime()
    },
    endpoints: {
      dve_nodes: "/api/dve-nodes",
      validation_tasks: "/api/validation-tasks",
      cognitive_engine: "/api/cognitive-engine",
      tee_security: "/api/tee-security",
      nrn_staking: "/api/nrn-staking",
      system_health: "/api/system-health"
    }
  };

  return NextResponse.json(healthStatus);
}