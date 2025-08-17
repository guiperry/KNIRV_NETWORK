import { beforeAll, afterAll, beforeEach, afterEach } from 'vitest'
import { setupServer } from 'msw/node'
import { rest } from 'msw'

// Mock server for API testing
const server = setupServer(
  // Mock player API endpoints
  rest.get('/api/players/:id', (req, res, ctx) => {
    const { id } = req.params
    return res(
      ctx.json({
        id,
        name: `Player ${id}`,
        nrnBalance: 1000.0,
        reputation: 85,
        color: '#FF6B6B',
        agents: []
      })
    )
  }),

  rest.post('/api/players', (req, res, ctx) => {
    return res(
      ctx.json({
        id: 'new_player_001',
        name: 'New Player',
        nrnBalance: 500.0,
        reputation: 0,
        color: '#4ECDC4',
        agents: []
      })
    )
  }),

  // Mock game state endpoints
  rest.get('/api/game/state', (req, res, ctx) => {
    return res(
      ctx.json({
        phase: 'active',
        players: [],
        errorNodes: [
          {
            id: 'error_001',
            position: { x: 10, y: 5, z: 0 },
            type: 'error',
            difficulty: 3,
            bounty: 50.0,
            isBeingResolved: false,
            description: 'Memory leak in game loop',
            progress: 0
          }
        ],
        skillNodes: [
          {
            id: 'skill_001',
            position: { x: 0, y: 0, z: 0 },
            type: 'skill',
            name: 'Advanced Debugging',
            category: 'debugging',
            createdBy: 'player_001',
            usageCount: 15,
            value: 25.0
          }
        ],
        currentPlayerId: 'player_001',
        timeRemaining: 1800,
        networkActivity: 75
      })
    )
  }),

  rest.post('/api/game/join', (req, res, ctx) => {
    return res(
      ctx.json({
        success: true,
        playerId: 'player_001',
        gameId: 'game_001'
      })
    )
  }),

  // Mock skill endpoints
  rest.get('/api/skills', (req, res, ctx) => {
    return res(
      ctx.json([
        {
          id: 'skill_001',
          name: 'Advanced Debugging',
          category: 'debugging',
          difficulty: 5,
          price: 75.0,
          description: 'Advanced debugging techniques'
        },
        {
          id: 'skill_002',
          name: 'Performance Optimization',
          category: 'optimization',
          difficulty: 7,
          price: 120.0,
          description: 'Code optimization strategies'
        }
      ])
    )
  }),

  rest.post('/api/skills/:id/purchase', (req, res, ctx) => {
    const { id } = req.params
    return res(
      ctx.json({
        success: true,
        skillId: id,
        newBalance: 925.0,
        transactionId: 'tx_001'
      })
    )
  }),

  // Mock error resolution endpoints
  rest.post('/api/errors/:id/resolve', (req, res, ctx) => {
    const { id } = req.params
    return res(
      ctx.json({
        success: true,
        errorId: id,
        reward: 50.0,
        experience: 100,
        newBalance: 1050.0
      })
    )
  }),

  // Mock agent endpoints
  rest.post('/api/agents/deploy', (req, res, ctx) => {
    return res(
      ctx.json({
        success: true,
        agentId: 'agent_001',
        deploymentId: 'deploy_001',
        estimatedCompletion: Date.now() + 30000 // 30 seconds
      })
    )
  }),

  rest.get('/api/agents/:id/status', (req, res, ctx) => {
    const { id } = req.params
    return res(
      ctx.json({
        id,
        status: 'active',
        currentTask: 'resolving_error_001',
        progress: 75,
        energy: 85,
        position: { x: 5, y: 2, z: 1 }
      })
    )
  }),

  // Mock leaderboard endpoints
  rest.get('/api/leaderboard', (req, res, ctx) => {
    return res(
      ctx.json([
        {
          playerId: 'player_001',
          playerName: 'TopPlayer',
          score: 2500,
          errorsResolved: 45,
          skillsCreated: 12,
          reputation: 95
        },
        {
          playerId: 'player_002',
          playerName: 'SecondPlace',
          score: 2200,
          errorsResolved: 38,
          skillsCreated: 8,
          reputation: 88
        }
      ])
    )
  }),

  // Mock statistics endpoints
  rest.get('/api/stats/player/:id', (req, res, ctx) => {
    const { id } = req.params
    return res(
      ctx.json({
        playerId: id,
        gamesPlayed: 42,
        errorsResolved: 156,
        skillsCreated: 23,
        totalEarnings: 2450.75,
        averageRating: 4.7,
        achievements: [
          'first_error_resolved',
          'skill_creator',
          'top_performer'
        ]
      })
    )
  }),

  // Mock WebSocket connection (for testing purposes)
  rest.get('/api/ws/connect', (req, res, ctx) => {
    return res(
      ctx.json({
        success: true,
        wsUrl: 'ws://localhost:3001',
        token: 'mock_ws_token'
      })
    )
  }),

  // Error handlers for testing error scenarios
  rest.get('/api/error/500', (req, res, ctx) => {
    return res(ctx.status(500), ctx.json({ error: 'Internal Server Error' }))
  }),

  rest.get('/api/error/404', (req, res, ctx) => {
    return res(ctx.status(404), ctx.json({ error: 'Not Found' }))
  }),

  rest.get('/api/error/timeout', (req, res, ctx) => {
    return res(ctx.delay(10000), ctx.json({ data: 'This will timeout' }))
  })
)

// Mock WebSocket for integration tests
class MockWebSocket {
  static CONNECTING = 0
  static OPEN = 1
  static CLOSING = 2
  static CLOSED = 3

  readyState = MockWebSocket.CONNECTING
  url: string
  onopen: ((event: Event) => void) | null = null
  onclose: ((event: CloseEvent) => void) | null = null
  onmessage: ((event: MessageEvent) => void) | null = null
  onerror: ((event: Event) => void) | null = null

  constructor(url: string) {
    this.url = url
    
    // Simulate connection opening
    setTimeout(() => {
      this.readyState = MockWebSocket.OPEN
      if (this.onopen) {
        this.onopen(new Event('open'))
      }
    }, 100)
  }

  send(data: string) {
    if (this.readyState !== MockWebSocket.OPEN) {
      throw new Error('WebSocket is not open')
    }
    
    // Echo back the message for testing
    setTimeout(() => {
      if (this.onmessage) {
        this.onmessage(new MessageEvent('message', { data }))
      }
    }, 50)
  }

  close() {
    this.readyState = MockWebSocket.CLOSING
    setTimeout(() => {
      this.readyState = MockWebSocket.CLOSED
      if (this.onclose) {
        this.onclose(new CloseEvent('close'))
      }
    }, 50)
  }

  addEventListener(type: string, listener: EventListener) {
    if (type === 'open') this.onopen = listener as any
    if (type === 'close') this.onclose = listener as any
    if (type === 'message') this.onmessage = listener as any
    if (type === 'error') this.onerror = listener as any
  }

  removeEventListener(type: string, listener: EventListener) {
    if (type === 'open') this.onopen = null
    if (type === 'close') this.onclose = null
    if (type === 'message') this.onmessage = null
    if (type === 'error') this.onerror = null
  }
}

beforeAll(() => {
  // Start the mock server
  server.listen({ onUnhandledRequest: 'error' })
  
  // Replace WebSocket with mock
  global.WebSocket = MockWebSocket as any
  
  // Set integration test environment variables
  process.env.VITE_API_URL = 'http://localhost:3000/api'
  process.env.VITE_WS_URL = 'ws://localhost:3001'
  process.env.NODE_ENV = 'test'
})

afterAll(() => {
  // Clean up
  server.close()
})

beforeEach(() => {
  // Reset any runtime request handlers
  server.resetHandlers()
})

afterEach(() => {
  // Clean up any test-specific state
})
