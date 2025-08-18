# Mock Data Audit Report
## Phase 2.3: Mock Removal and Real Implementation

**Date:** 2025-01-16  
**Objective:** Identify all hard-coded mock data across frontend and backend for replacement with real API calls  
**Scope:** Complete codebase analysis for mock data patterns

## Executive Summary

This audit identifies **23 instances** of hard-coded mock data across the Agentic-Engine frontend that need to be replaced with real API calls. The mock data is primarily concentrated in dashboard components, analytics, and fallback scenarios.

## Mock Data Categories

### 1. Dashboard Statistics (HIGH PRIORITY)
**Location:** `gui/src/components/Dashboard.tsx`
**Lines:** 29-34, 36-77, 79-85, 140-145

**Mock Data:**
```javascript
const stats = [
  { label: 'Active Agents', value: '47', change: '+12%', icon: Bot },
  { label: 'Target Systems', value: '23', change: '+8%', icon: Target },
  { label: 'Inferences Today', value: '1,847', change: '+34%', icon: Activity },
  { label: 'Success Rate', value: '97.2%', change: '+1.8%', icon: TrendingUp },
];

const recentActivity = [
  { id: 1, type: 'inference', agent: 'CyberPunk Agent #7804', ... },
  { id: 2, type: 'deployment', agent: 'Data Miner #3749', ... },
  // ... 4 total activity items
];

const targetSystems = [
  { name: 'Chrome Browser', status: 'connected', agents: 8, ... },
  { name: 'Local File System', status: 'connected', agents: 12, ... },
  // ... 5 total target systems
];

const mockTargetSystems = [
  { id: 1, name: 'Production Server', type: 'Linux Server' },
  // ... 4 total mock targets
];
```

**Impact:** Core dashboard functionality relies entirely on static data
**Replacement:** Real-time API calls to analytics and system status endpoints

### 2. Analytics Dashboard (HIGH PRIORITY)
**Location:** `gui/src/components/analytics/AnalyticsDashboard.jsx`
**Lines:** 38-56

**Mock Data:**
```javascript
setAnalyticsData({
  summary: {
    totalWorkflows: 2847,
    completedWorkflows: 2765,
    failedWorkflows: 82,
    successRate: 97.2,
    averageResponseTime: 4.2,
    todayWorkflows: 124,
    agentCount: 47,
    targetCount: 23
  },
  chartData: [
    { name: 'Jan', orchestrations: 420, success: 96.2 },
    // ... 6 months of data
  ]
});
```

**Impact:** Analytics page shows fake performance metrics
**Replacement:** Real analytics API with time-series data

### 3. Target Manager Demo Data (MEDIUM PRIORITY)
**Location:** `gui/src/components/TargetManager.jsx`
**Lines:** 53-149

**Mock Data:**
```javascript
const demoTargets = [
  {
    id: 1,
    name: 'Chrome Browser',
    type: 'browser',
    status: 'connected',
    activeAgents: 8,
    capabilities: ['Web Scraping', 'DOM Analysis', ...],
    // ... extensive target configuration
  },
  // ... 5 total demo targets
];
```

**Impact:** Target discovery shows fake systems when real discovery fails
**Replacement:** Enhanced real target discovery with proper error handling

### 4. Agent Manager Fallback Data (MEDIUM PRIORITY)
**Location:** `gui/src/components/AgentManager.jsx`
**Lines:** 151-169

**Mock Data:**
```javascript
// Fallback to sample data for demo purposes
setAgents([
  {
    id: '1',
    name: 'CyberPunk Agent #7804',
    collection: 'CyberPunk Collective',
    status: 'idle',
    capabilities: ['Web Analysis', 'Data Extraction', 'Content Monitoring'],
    // ... agent configuration
  }
]);
```

**Impact:** Shows fake agent when API fails
**Replacement:** Proper error handling without fallback data

### 5. Web Connections Sample Data (MEDIUM PRIORITY)
**Location:** `gui/src/components/WebConnections.jsx`
**Lines:** 106-174

**Mock Data:**
```javascript
setConnections([
  {
    id: '1',
    name: 'Salesforce CRM',
    provider: 'Salesforce',
    status: 'connected',
    scopes: ['read', 'write', 'contacts', 'opportunities'],
    // ... 6 total connections
  }
]);
```

**Impact:** Shows fake external service connections
**Replacement:** Real connection management API

### 6. MCP Capability Manager (LOW PRIORITY)
**Location:** `gui/src/components/MCPCapabilityManager.jsx`
**Lines:** 114-124

**Mock Data:**
```javascript
const newCapability = {
  id: `mcp-${serverId}`,
  name: serverName,
  provider: 'MCP Server',
  type: 'mcp_capability',
  status: 'available',
  // ... capability configuration
};
```

**Impact:** Creates fake capabilities during MCP server transformation
**Replacement:** Real MCP server capability discovery

### 7. Test Mock Data (LOW PRIORITY)
**Locations:** 
- `src/services/__mocks__/targetDiscovery.js`
- `gui/src/tests/AgentManager.test.jsx`
- `gui/src/tests/MCPCapabilityManager.test.jsx`
- `gui/test/setup.js`

**Impact:** Test-only mock data (acceptable for testing)
**Action:** Keep for testing, ensure not used in production

## Backend Mock Data

### 8. Demo Mode Configuration (MEDIUM PRIORITY)
**Location:** Various API endpoints
**Pattern:** Demo data toggle functionality

**Current Implementation:**
- Demo data status endpoint: `/api/v1/debug/demo-data-status`
- Demo data mixed with real data in target discovery
- Conditional demo data loading based on API response

**Impact:** Demo mode uses same code paths but with seeded data
**Status:** Partially implemented correctly

## Replacement Strategy

### Phase 1: Dashboard Analytics (Week 1)
**Priority:** HIGH
**Components:** Dashboard.tsx, AnalyticsDashboard.jsx
**API Endpoints Needed:**
- `GET /api/v1/analytics/dashboard-stats`
- `GET /api/v1/analytics/recent-activity`
- `GET /api/v1/analytics/time-series`
- `GET /api/v1/system/status`

### Phase 2: Target and Agent Management (Week 2)
**Priority:** MEDIUM
**Components:** TargetManager.jsx, AgentManager.jsx
**API Endpoints Needed:**
- Enhanced error handling for existing endpoints
- Remove fallback mock data
- Implement proper loading states

### Phase 3: External Integrations (Week 3)
**Priority:** MEDIUM
**Components:** WebConnections.jsx, MCPCapabilityManager.jsx
**API Endpoints Needed:**
- `GET /api/v1/connections`
- `POST /api/v1/connections/test`
- Enhanced MCP capability discovery

## Implementation Requirements

### New API Endpoints

#### Dashboard Analytics API
```go
// GET /api/v1/analytics/dashboard-stats
type DashboardStats struct {
    ActiveAgents    int     `json:"active_agents"`
    TargetSystems   int     `json:"target_systems"`
    InferencesToday int     `json:"inferences_today"`
    SuccessRate     float64 `json:"success_rate"`
    Changes         map[string]string `json:"changes"`
}

// GET /api/v1/analytics/recent-activity
type RecentActivity struct {
    ID         string `json:"id"`
    Type       string `json:"type"`
    Agent      string `json:"agent"`
    Target     string `json:"target"`
    Capability string `json:"capability"`
    Status     string `json:"status"`
    Time       string `json:"time"`
    Result     string `json:"result,omitempty"`
}
```

#### System Status API
```go
// GET /api/v1/system/status
type SystemStatus struct {
    TargetSystems []TargetSystemStatus `json:"target_systems"`
    Timestamp     time.Time            `json:"timestamp"`
}

type TargetSystemStatus struct {
    Name         string `json:"name"`
    Status       string `json:"status"`
    Agents       int    `json:"agents"`
    LastActivity string `json:"last_activity"`
    Type         string `json:"type"`
}
```

### Frontend Changes

#### Remove Mock Data Constants
1. Delete all `const mockData = [...]` arrays
2. Replace with API calls in `useEffect` hooks
3. Add proper loading states and error handling
4. Implement retry logic for failed API calls

#### Error Handling Strategy
```javascript
// Replace fallback mock data with proper error states
const [data, setData] = useState([]);
const [loading, setLoading] = useState(true);
const [error, setError] = useState(null);

useEffect(() => {
  const fetchData = async () => {
    try {
      const response = await api.get('/endpoint');
      setData(response.data);
      setError(null);
    } catch (err) {
      setError(err.message);
      // Don't set mock data - show error state instead
    } finally {
      setLoading(false);
    }
  };
  
  fetchData();
}, []);
```

## Demo Mode Refactoring

### Current Demo Mode Issues
1. **Mixed Data Sources:** Demo data mixed with real data
2. **Inconsistent Patterns:** Different components handle demo mode differently
3. **Code Duplication:** Demo data duplicated across components

### Proposed Demo Mode Architecture
```javascript
// Centralized demo data service
class DemoDataService {
  constructor(apiService) {
    this.api = apiService;
    this.isDemoMode = false;
  }
  
  async getDashboardStats() {
    if (this.isDemoMode) {
      return this.generateDemoStats();
    }
    return this.api.getDashboardStats();
  }
  
  generateDemoStats() {
    // Generate realistic demo data with timestamps
    return {
      activeAgents: Math.floor(Math.random() * 50) + 20,
      targetSystems: Math.floor(Math.random() * 30) + 10,
      // ... other demo data
    };
  }
}
```

## Success Metrics

### Completion Criteria
1. **Zero Hard-coded Arrays:** No `const mockData = [...]` in production components
2. **Real API Integration:** All dashboard data from live APIs
3. **Proper Error Handling:** No fallback to mock data on API failures
4. **Demo Mode Consistency:** Unified demo data generation approach
5. **Performance Maintained:** API response times < 500ms
6. **Test Coverage:** All new API endpoints covered by tests

### Validation Steps
1. **Code Review:** Search for remaining mock data patterns
2. **API Testing:** Verify all endpoints return expected data structures
3. **Error Testing:** Confirm proper error states without mock fallbacks
4. **Demo Mode Testing:** Verify demo mode generates realistic data
5. **Performance Testing:** Measure API response times under load

## Timeline

**Week 1:** Dashboard Analytics Implementation
**Week 2:** Target and Agent Management Cleanup  
**Week 3:** External Integrations and Demo Mode Refactoring
**Week 4:** Testing, Validation, and Documentation

## Conclusion

The mock data audit reveals a systematic pattern of hard-coded data that needs replacement with real API calls. The highest priority items are dashboard statistics and analytics, which directly impact user experience. The proposed phased approach ensures minimal disruption while achieving complete mock data removal.

**Next Steps:**
1. Begin Dashboard Analytics API implementation
2. Create centralized demo data service
3. Implement proper error handling patterns
4. Remove all hard-coded mock arrays

---
*Audit completed as part of Phase 2: Data Consistency and Validation implementation.*
