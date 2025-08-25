import { NextRequest, NextResponse } from 'next/server';

// Required for static export
export const dynamic = 'force-static';
export const revalidate = false;

interface DVENode {
  id: string;
  name: string;
  status: "online" | "offline" | "maintenance";
  cpu_usage: number;
  memory_usage: number;
  tee_type: "SGX" | "SEV-SNP" | "TDX" | "None";
  stake_amount: number;
  reputation_score: number;
  last_heartbeat: string;
  location?: string;
  ip_address?: string;
}

// Mock data for DVE nodes
const mockDVENodes: DVENode[] = [
  {
    id: "dve-001",
    name: "KNIRV-Node-Alpha",
    status: "online",
    cpu_usage: 45,
    memory_usage: 62,
    tee_type: "SGX",
    stake_amount: 50000,
    reputation_score: 98,
    last_heartbeat: new Date().toISOString(),
    location: "US-East-1",
    ip_address: "192.168.1.100"
  },
  {
    id: "dve-002",
    name: "KNIRV-Node-Beta",
    status: "online",
    cpu_usage: 78,
    memory_usage: 85,
    tee_type: "SEV-SNP",
    stake_amount: 75000,
    reputation_score: 95,
    last_heartbeat: new Date().toISOString(),
    location: "EU-West-1",
    ip_address: "192.168.1.101"
  },
  {
    id: "dve-003",
    name: "KNIRV-Node-Gamma",
    status: "maintenance",
    cpu_usage: 0,
    memory_usage: 0,
    tee_type: "TDX",
    stake_amount: 60000,
    reputation_score: 92,
    last_heartbeat: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
    location: "Asia-Pacific-1",
    ip_address: "192.168.1.102"
  }
];

export async function GET(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url);
    const status = searchParams.get('status');
    const tee_type = searchParams.get('tee_type');
    
    let filteredNodes = [...mockDVENodes];
    
    if (status) {
      filteredNodes = filteredNodes.filter(node => node.status === status);
    }
    
    if (tee_type) {
      filteredNodes = filteredNodes.filter(node => node.tee_type === tee_type);
    }
    
    return NextResponse.json({
      success: true,
      data: filteredNodes,
      total: filteredNodes.length,
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    return NextResponse.json(
      { success: false, error: 'Failed to fetch DVE nodes' },
      { status: 500 }
    );
  }
}

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();
    const { name, tee_type, stake_amount, location, ip_address } = body;
    
    if (!name || !tee_type || !stake_amount) {
      return NextResponse.json(
        { success: false, error: 'Missing required fields' },
        { status: 400 }
      );
    }
    
    const newNode: DVENode = {
      id: `dve-${String(mockDVENodes.length + 1).padStart(3, '0')}`,
      name,
      status: "online",
      cpu_usage: 0,
      memory_usage: 0,
      tee_type,
      stake_amount,
      reputation_score: 100,
      last_heartbeat: new Date().toISOString(),
      location,
      ip_address
    };
    
    mockDVENodes.push(newNode);
    
    return NextResponse.json({
      success: true,
      data: newNode,
      message: 'DVE node created successfully',
      timestamp: new Date().toISOString()
    }, { status: 201 });
  } catch (error) {
    return NextResponse.json(
      { success: false, error: 'Failed to create DVE node' },
      { status: 500 }
    );
  }
}