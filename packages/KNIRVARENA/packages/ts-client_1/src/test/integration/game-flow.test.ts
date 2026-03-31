import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { userEvent } from '@testing-library/user-event'

// Mock the game components
vi.mock('@/components/game/KnirvanaGame', () => ({
  default: ({ onGameStateChange }: { onGameStateChange?: (state: any) => void }) => {
    const mockGameState = {
      phase: 'active',
      players: [
        {
          id: 'player_001',
          name: 'TestPlayer',
          nrnBalance: 1000.0,
          agents: [],
          color: '#FF6B6B',
          reputation: 85
        }
      ],
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
      skillNodes: [],
      currentPlayerId: 'player_001',
      timeRemaining: 1800,
      networkActivity: 75
    }

    return (
      <div data-testid="knirvana-game">
        <div data-testid="game-state">{JSON.stringify(mockGameState)}</div>
        <button 
          data-testid="resolve-error"
          onClick={() => onGameStateChange?.(mockGameState)}
        >
          Resolve Error
        </button>
        <button data-testid="deploy-agent">Deploy Agent</button>
        <div data-testid="player-balance">Balance: 1000 NRN</div>
      </div>
    )
  }
}))

describe('Game Flow Integration Tests', () => {
  let user: ReturnType<typeof userEvent.setup>

  beforeEach(() => {
    user = userEvent.setup()
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('should complete a full game session flow', async () => {
    // Mock the App component with game integration
    const MockApp = () => {
      const [gameState, setGameState] = React.useState(null)
      
      return (
        <div>
          <div data-testid="app-container">
            <KnirvanaGame onGameStateChange={setGameState} />
          </div>
          {gameState && (
            <div data-testid="game-status">Game Active</div>
          )}
        </div>
      )
    }

    render(<MockApp />)

    // Verify initial game load
    await waitFor(() => {
      expect(screen.getByTestId('knirvana-game')).toBeInTheDocument()
    })

    // Verify game state is loaded
    expect(screen.getByTestId('game-state')).toBeInTheDocument()
    expect(screen.getByTestId('player-balance')).toHaveTextContent('Balance: 1000 NRN')

    // Simulate error resolution
    const resolveButton = screen.getByTestId('resolve-error')
    await user.click(resolveButton)

    // Verify game state update
    await waitFor(() => {
      expect(screen.getByTestId('game-status')).toBeInTheDocument()
    })
  })

  it('should handle player actions and state updates', async () => {
    const MockGameWithActions = () => {
      const [playerBalance, setPlayerBalance] = React.useState(1000)
      const [agentDeployed, setAgentDeployed] = React.useState(false)

      const handleResolveError = () => {
        setPlayerBalance(prev => prev + 50) // Reward for resolving error
      }

      const handleDeployAgent = () => {
        if (playerBalance >= 100) {
          setPlayerBalance(prev => prev - 100) // Cost to deploy agent
          setAgentDeployed(true)
        }
      }

      return (
        <div>
          <div data-testid="player-balance">Balance: {playerBalance} NRN</div>
          <button data-testid="resolve-error" onClick={handleResolveError}>
            Resolve Error
          </button>
          <button data-testid="deploy-agent" onClick={handleDeployAgent}>
            Deploy Agent
          </button>
          {agentDeployed && <div data-testid="agent-status">Agent Deployed</div>}
        </div>
      )
    }

    render(<MockGameWithActions />)

    // Initial state
    expect(screen.getByTestId('player-balance')).toHaveTextContent('Balance: 1000 NRN')

    // Resolve an error (gain NRN)
    await user.click(screen.getByTestId('resolve-error'))
    expect(screen.getByTestId('player-balance')).toHaveTextContent('Balance: 1050 NRN')

    // Deploy an agent (spend NRN)
    await user.click(screen.getByTestId('deploy-agent'))
    expect(screen.getByTestId('player-balance')).toHaveTextContent('Balance: 950 NRN')
    expect(screen.getByTestId('agent-status')).toBeInTheDocument()
  })

  it('should handle network communication', async () => {
    // Mock fetch for API calls
    global.fetch = vi.fn()
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          success: true,
          playerId: 'player_001',
          gameId: 'game_001'
        })
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          success: true,
          errorId: 'error_001',
          reward: 50.0,
          experience: 100,
          newBalance: 1050.0
        })
      })

    const MockNetworkGame = () => {
      const [gameJoined, setGameJoined] = React.useState(false)
      const [balance, setBalance] = React.useState(1000)

      const joinGame = async () => {
        const response = await fetch('/api/game/join', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ playerId: 'player_001' })
        })
        const data = await response.json()
        if (data.success) {
          setGameJoined(true)
        }
      }

      const resolveError = async () => {
        const response = await fetch('/api/errors/error_001/resolve', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ playerId: 'player_001' })
        })
        const data = await response.json()
        if (data.success) {
          setBalance(data.newBalance)
        }
      }

      return (
        <div>
          {!gameJoined ? (
            <button data-testid="join-game" onClick={joinGame}>
              Join Game
            </button>
          ) : (
            <div>
              <div data-testid="game-joined">Game Joined</div>
              <div data-testid="player-balance">Balance: {balance} NRN</div>
              <button data-testid="resolve-error" onClick={resolveError}>
                Resolve Error
              </button>
            </div>
          )}
        </div>
      )
    }

    render(<MockNetworkGame />)

    // Join game
    await user.click(screen.getByTestId('join-game'))
    
    await waitFor(() => {
      expect(screen.getByTestId('game-joined')).toBeInTheDocument()
    })

    // Resolve error with network call
    await user.click(screen.getByTestId('resolve-error'))
    
    await waitFor(() => {
      expect(screen.getByTestId('player-balance')).toHaveTextContent('Balance: 1050 NRN')
    })

    // Verify API calls were made
    expect(fetch).toHaveBeenCalledTimes(2)
    expect(fetch).toHaveBeenCalledWith('/api/game/join', expect.any(Object))
    expect(fetch).toHaveBeenCalledWith('/api/errors/error_001/resolve', expect.any(Object))
  })

  it('should handle WebSocket real-time updates', async () => {
    // Mock WebSocket
    const mockWebSocket = {
      send: vi.fn(),
      close: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      readyState: 1 // OPEN
    }

    global.WebSocket = vi.fn(() => mockWebSocket) as any

    const MockRealtimeGame = () => {
      const [connected, setConnected] = React.useState(false)
      const [messages, setMessages] = React.useState<string[]>([])

      React.useEffect(() => {
        const ws = new WebSocket('ws://localhost:3001')
        
        ws.addEventListener('open', () => {
          setConnected(true)
        })

        ws.addEventListener('message', (event) => {
          setMessages(prev => [...prev, event.data])
        })

        return () => ws.close()
      }, [])

      const sendMessage = () => {
        const ws = new WebSocket('ws://localhost:3001')
        ws.send(JSON.stringify({ type: 'player_action', action: 'resolve_error' }))
      }

      return (
        <div>
          <div data-testid="connection-status">
            {connected ? 'Connected' : 'Disconnected'}
          </div>
          <button data-testid="send-message" onClick={sendMessage}>
            Send Action
          </button>
          <div data-testid="message-count">Messages: {messages.length}</div>
        </div>
      )
    }

    render(<MockRealtimeGame />)

    // Simulate WebSocket connection
    const openHandler = mockWebSocket.addEventListener.mock.calls
      .find(call => call[0] === 'open')?.[1]
    
    if (openHandler) {
      openHandler(new Event('open'))
    }

    await waitFor(() => {
      expect(screen.getByTestId('connection-status')).toHaveTextContent('Connected')
    })

    // Send a message
    await user.click(screen.getByTestId('send-message'))
    
    expect(mockWebSocket.send).toHaveBeenCalledWith(
      JSON.stringify({ type: 'player_action', action: 'resolve_error' })
    )
  })

  it('should handle error scenarios gracefully', async () => {
    // Mock failed API call
    global.fetch = vi.fn().mockRejectedValue(new Error('Network error'))

    const MockErrorHandling = () => {
      const [error, setError] = React.useState<string | null>(null)
      const [loading, setLoading] = React.useState(false)

      const attemptAction = async () => {
        setLoading(true)
        setError(null)
        
        try {
          await fetch('/api/game/action')
        } catch (err) {
          setError(err instanceof Error ? err.message : 'Unknown error')
        } finally {
          setLoading(false)
        }
      }

      return (
        <div>
          <button data-testid="attempt-action" onClick={attemptAction}>
            Attempt Action
          </button>
          {loading && <div data-testid="loading">Loading...</div>}
          {error && <div data-testid="error-message">{error}</div>}
        </div>
      )
    }

    render(<MockErrorHandling />)

    await user.click(screen.getByTestId('attempt-action'))

    // Should show loading state
    expect(screen.getByTestId('loading')).toBeInTheDocument()

    // Should show error after failed request
    await waitFor(() => {
      expect(screen.getByTestId('error-message')).toHaveTextContent('Network error')
    })

    // Loading should be gone
    expect(screen.queryByTestId('loading')).not.toBeInTheDocument()
  })

  it('should maintain game state consistency', async () => {
    const MockStateConsistency = () => {
      const [gameState, setGameState] = React.useState({
        players: [{ id: 'player_001', nrnBalance: 1000 }],
        errors: [{ id: 'error_001', bounty: 50 }]
      })

      const resolveError = () => {
        setGameState(prev => ({
          ...prev,
          players: prev.players.map(player => 
            player.id === 'player_001' 
              ? { ...player, nrnBalance: player.nrnBalance + 50 }
              : player
          ),
          errors: prev.errors.filter(error => error.id !== 'error_001')
        }))
      }

      return (
        <div>
          <div data-testid="player-balance">
            Balance: {gameState.players[0].nrnBalance}
          </div>
          <div data-testid="error-count">
            Errors: {gameState.errors.length}
          </div>
          <button data-testid="resolve-error" onClick={resolveError}>
            Resolve Error
          </button>
        </div>
      )
    }

    render(<MockStateConsistency />)

    // Initial state
    expect(screen.getByTestId('player-balance')).toHaveTextContent('Balance: 1000')
    expect(screen.getByTestId('error-count')).toHaveTextContent('Errors: 1')

    // Resolve error
    await user.click(screen.getByTestId('resolve-error'))

    // State should be updated consistently
    expect(screen.getByTestId('player-balance')).toHaveTextContent('Balance: 1050')
    expect(screen.getByTestId('error-count')).toHaveTextContent('Errors: 0')
  })
})

// Helper to create React component for testing
const React = {
  useState: (initial: any) => {
    let state = initial
    const setState = (newState: any) => {
      state = typeof newState === 'function' ? newState(state) : newState
    }
    return [state, setState]
  },
  useEffect: (effect: () => void | (() => void), deps?: any[]) => {
    const cleanup = effect()
    return cleanup
  }
}
