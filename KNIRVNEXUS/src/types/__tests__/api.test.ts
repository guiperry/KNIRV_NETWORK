import type {
  APIResponse,
  DVENode,
  Agent,
  AgentResourceLimits,
  DNSRecord,
  DVERentalPlan,
  ValidationTask,
  SystemHealth,
  UserProfile,
} from '../api';

describe('API Types', () => {
  describe('APIResponse', () => {
    it('should have correct structure for success response', () => {
      const response: APIResponse<string> = {
        success: true,
        data: 'test data',
        timestamp: new Date().toISOString(),
      };

      expect(response.success).toBe(true);
      expect(response.data).toBe('test data');
      expect(response.timestamp).toBeDefined();
      expect(response.error).toBeUndefined();
    });

    it('should have correct structure for error response', () => {
      const response: APIResponse = {
        success: false,
        error: 'Test error message',
        timestamp: new Date().toISOString(),
      };

      expect(response.success).toBe(false);
      expect(response.error).toBe('Test error message');
      expect(response.timestamp).toBeDefined();
      expect(response.data).toBeUndefined();
    });
  });

  describe('DVENode', () => {
    it('should have all required fields', () => {
      const node: DVENode = {
        id: 'node-1',
        name: 'Test Node',
        status: 'online',
        tee_type: 'sgx',
        stake_amount: 1000,
        reputation_score: 95,
        location: 'US-East',
        ip_address: '192.168.1.1',
        public_key: 'test-public-key',
        capabilities: ['validation', 'inference'],
        last_heartbeat: '2024-01-01T00:00:00Z',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
        cpu_usage: 50.5,
        memory_usage: 75.2,
        network_latency: 10,
      };

      expect(node.id).toBe('node-1');
      expect(node.status).toBe('online');
      expect(node.tee_type).toBe('sgx');
      expect(typeof node.stake_amount).toBe('number');
      expect(Array.isArray(node.capabilities)).toBe(true);
    });

    it('should allow optional geographic coordinates', () => {
      const node: DVENode = {
        id: 'node-1',
        name: 'Test Node',
        status: 'online',
        tee_type: 'sgx',
        stake_amount: 1000,
        reputation_score: 95,
        location: 'US-East',
        ip_address: '192.168.1.1',
        public_key: 'test-public-key',
        capabilities: ['validation'],
        last_heartbeat: '2024-01-01T00:00:00Z',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
        cpu_usage: 50.5,
        memory_usage: 75.2,
        network_latency: 10,
        latitude: 40.7128,
        longitude: -74.0060,
      };

      expect(node.latitude).toBe(40.7128);
      expect(node.longitude).toBe(-74.0060);
    });
  });

  describe('Agent', () => {
    it('should have all required fields', () => {
      const agent: Agent = {
        id: 'agent-1',
        name: 'Test Agent',
        description: 'Test description',
        version: '1.0.0',
        author: 'Test Author',
        type: 'WASM',
        status: 'uploaded',
        file_path: '/path/to/agent.wasm',
        file_size: 1024,
        file_hash: 'sha256-hash',
        capabilities: ['inference'],
        dependencies: [],
        configuration: {},
        metadata: {},
        tags: ['test'],
        uploaded_at: '2024-01-01T00:00:00Z',
        last_modified: '2024-01-01T00:00:00Z',
        uploaded_by: 'user-1',
      };

      expect(agent.id).toBe('agent-1');
      expect(agent.type).toBe('WASM');
      expect(agent.status).toBe('uploaded');
      expect(Array.isArray(agent.capabilities)).toBe(true);
      expect(typeof agent.configuration).toBe('object');
    });
  });

  describe('AgentResourceLimits', () => {
    it('should have correct resource limit structure', () => {
      const limits: AgentResourceLimits = {
        max_memory_mb: 512,
        max_cpu_percent: 50,
        max_execution_time_seconds: 300,
        max_concurrency: 10,
        max_disk_mb: 1024,
        network_access: true,
        file_system_access: false,
      };

      expect(typeof limits.max_memory_mb).toBe('number');
      expect(typeof limits.network_access).toBe('boolean');
      expect(typeof limits.file_system_access).toBe('boolean');
    });
  });

  describe('DNSRecord', () => {
    it('should have all required fields', () => {
      const record: DNSRecord = {
        id: 'dns-1',
        name: 'test.example.com',
        type: 'A',
        value: '192.168.1.1',
        ttl: 300,
        zone: 'example.com',
      };

      expect(record.id).toBe('dns-1');
      expect(record.type).toBe('A');
      expect(typeof record.ttl).toBe('number');
    });

    it('should allow optional fields', () => {
      const record: DNSRecord = {
        id: 'dns-1',
        name: 'test.example.com',
        type: 'A',
        value: '192.168.1.1',
        ttl: 300,
        zone: 'example.com',
        proxied: true,
        priority: 10,
        comment: 'Test record',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      };

      expect(record.proxied).toBe(true);
      expect(record.priority).toBe(10);
      expect(record.comment).toBe('Test record');
    });
  });

  describe('DVERentalPlan', () => {
    it('should have correct rental plan structure', () => {
      const plan: DVERentalPlan = {
        id: 'plan-1',
        name: 'Basic Plan',
        description: 'Basic DVE rental plan',
        price_per_hour: 10,
        cpu_cores: 2,
        memory_gb: 4,
        disk_gb: 50,
        bandwidth_mbps: 100,
        features: ['basic-support'],
      };

      expect(plan.id).toBe('plan-1');
      expect(typeof plan.price_per_hour).toBe('number');
      expect(Array.isArray(plan.features)).toBe(true);
    });
  });

  describe('ValidationTask', () => {
    it('should have all required fields', () => {
      const task: ValidationTask = {
        id: 'task-1',
        type: 'skillnode',
        status: 'pending',
        priority: 5,
        test_cases: [],
        required_tee_type: 'sgx',
        requested_by: 'user-1',
        parameters: {},
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
        timeout_at: '2024-01-01T01:00:00Z',
      };

      expect(task.id).toBe('task-1');
      expect(task.type).toBe('skillnode');
      expect(task.status).toBe('pending');
      expect(typeof task.priority).toBe('number');
    });

    it('should allow optional fields', () => {
      const task: ValidationTask = {
        id: 'task-1',
        type: 'skillnode',
        status: 'running',
        priority: 5,
        test_cases: [],
        required_tee_type: 'sgx',
        requested_by: 'user-1',
        parameters: {},
        completion_percentage: 50,
        estimated_completion_time: '2024-01-01T02:00:00Z',
        assigned_node_id: 'node-1',
        skill_code: 'test-skill',
        failure_context: 'test failure',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
        started_at: '2024-01-01T00:30:00Z',
        timeout_at: '2024-01-01T01:00:00Z',
      };

      expect(task.completion_percentage).toBe(50);
      expect(task.assigned_node_id).toBe('node-1');
      expect(task.skill_code).toBe('test-skill');
    });
  });

  describe('SystemHealth', () => {
    it('should have correct system health structure', () => {
      const health: SystemHealth = {
        status: 'healthy',
        timestamp: '2024-01-01T00:00:00Z',
        uptime: 86400,
        components: {
          database: {
            status: 'healthy',
            message: 'Database is operational',
          },
        },
        alerts: [],
      };

      expect(health.status).toBe('healthy');
      expect(typeof health.uptime).toBe('number');
      expect(typeof health.components).toBe('object');
      expect(Array.isArray(health.alerts)).toBe(true);
    });
  });

  describe('UserProfile', () => {
    it('should have all required fields', () => {
      const user: UserProfile = {
        id: 'user-1',
        email: 'test@example.com',
        username: 'testuser',
        role: 'admin',
        permissions: {
          can_manage_users: true,
          can_manage_nodes: true,
          can_manage_agents: true,
          can_view_metrics: true,
          can_manage_dns: true,
          can_access_admin_panel: true,
        },
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
        last_login: '2024-01-01T00:00:00Z',
        is_active: true,
      };

      expect(user.id).toBe('user-1');
      expect(user.role).toBe('admin');
      expect(typeof user.permissions).toBe('object');
      expect(typeof user.is_active).toBe('boolean');
    });
  });
});
