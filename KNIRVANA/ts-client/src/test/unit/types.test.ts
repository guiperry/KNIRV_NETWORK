import { describe, it, expect } from 'vitest'
import * as THREE from 'three'
import type { 
  ErrorNode, 
  SkillNode, 
  AIAgent, 
  Player, 
  GameState,
  Controls 
} from '@/types/game'

describe('Game Types', () => {
  describe('ErrorNode', () => {
    it('should create a valid ErrorNode', () => {
      const errorNode: ErrorNode = {
        id: 'error_001',
        position: new THREE.Vector3(10, 5, 0),
        type: 'error',
        difficulty: 3,
        bounty: 50.0,
        isBeingResolved: false,
        description: 'Memory leak in game loop',
        progress: 0
      }

      expect(errorNode.id).toBe('error_001')
      expect(errorNode.type).toBe('error')
      expect(errorNode.difficulty).toBe(3)
      expect(errorNode.bounty).toBe(50.0)
      expect(errorNode.isBeingResolved).toBe(false)
      expect(errorNode.progress).toBe(0)
      expect(errorNode.position).toBeInstanceOf(THREE.Vector3)
      expect(errorNode.position.x).toBe(10)
    })

    it('should handle ErrorNode with resolver', () => {
      const errorNode: ErrorNode = {
        id: 'error_002',
        position: new THREE.Vector3(-5, 8, 3),
        type: 'error',
        difficulty: 5,
        bounty: 100.0,
        isBeingResolved: true,
        resolvedBy: 'player_001',
        description: 'Race condition in networking module',
        progress: 25
      }

      expect(errorNode.isBeingResolved).toBe(true)
      expect(errorNode.resolvedBy).toBe('player_001')
      expect(errorNode.progress).toBe(25)
    })

    it('should validate difficulty range', () => {
      const createErrorNode = (difficulty: number) => ({
        id: 'test',
        position: new THREE.Vector3(),
        type: 'error' as const,
        difficulty,
        bounty: 50.0,
        isBeingResolved: false,
        description: 'Test error',
        progress: 0
      })

      // Valid difficulties
      expect(() => createErrorNode(1)).not.toThrow()
      expect(() => createErrorNode(5)).not.toThrow()
      expect(() => createErrorNode(10)).not.toThrow()

      // Edge cases
      const minDifficulty = createErrorNode(1)
      const maxDifficulty = createErrorNode(10)
      
      expect(minDifficulty.difficulty).toBe(1)
      expect(maxDifficulty.difficulty).toBe(10)
    })
  })

  describe('SkillNode', () => {
    it('should create a valid SkillNode', () => {
      const skillNode: SkillNode = {
        id: 'skill_001',
        position: new THREE.Vector3(0, 0, 0),
        type: 'skill',
        name: 'Advanced Debugging',
        category: 'debugging',
        createdBy: 'player_001',
        usageCount: 15,
        value: 25.0
      }

      expect(skillNode.id).toBe('skill_001')
      expect(skillNode.type).toBe('skill')
      expect(skillNode.name).toBe('Advanced Debugging')
      expect(skillNode.category).toBe('debugging')
      expect(skillNode.createdBy).toBe('player_001')
      expect(skillNode.usageCount).toBe(15)
      expect(skillNode.value).toBe(25.0)
    })

    it('should handle skill value calculations', () => {
      const calculateSkillValue = (usageCount: number, difficulty: number) => {
        const baseValue = 10
        const usageFactor = Math.sqrt(usageCount) * 2
        const difficultyFactor = difficulty * 5
        return baseValue + usageFactor + difficultyFactor
      }

      const lowUsageSkill = calculateSkillValue(1, 3)
      const highUsageSkill = calculateSkillValue(100, 3)
      
      expect(highUsageSkill).toBeGreaterThan(lowUsageSkill)
      expect(lowUsageSkill).toBeGreaterThan(0)
    })
  })

  describe('AIAgent', () => {
    it('should create a valid AIAgent', () => {
      const agent: AIAgent = {
        id: 'agent_001',
        position: new THREE.Vector3(5, 2, 1),
        targetPosition: new THREE.Vector3(10, 2, 1),
        type: 'resolver',
        owner: 'player_001',
        skills: ['javascript', 'rust', 'debugging'],
        isActive: true,
        energy: 100,
        maxEnergy: 100,
        specialization: 'debugging',
        thoughtProcess: ['Analyzing error...', 'Identifying root cause...']
      }

      expect(agent.id).toBe('agent_001')
      expect(agent.type).toBe('resolver')
      expect(agent.owner).toBe('player_001')
      expect(agent.skills).toHaveLength(3)
      expect(agent.skills).toContain('javascript')
      expect(agent.isActive).toBe(true)
      expect(agent.energy).toBe(100)
      expect(agent.maxEnergy).toBe(100)
      expect(agent.thoughtProcess).toHaveLength(2)
    })

    it('should handle different agent types', () => {
      const resolverAgent: AIAgent = {
        id: 'resolver_001',
        position: new THREE.Vector3(),
        targetPosition: new THREE.Vector3(),
        type: 'resolver',
        owner: 'player_001',
        skills: ['debugging'],
        isActive: true,
        energy: 100,
        maxEnergy: 100,
        specialization: 'error_resolution',
        thoughtProcess: []
      }

      const observerAgent: AIAgent = {
        id: 'observer_001',
        position: new THREE.Vector3(),
        targetPosition: new THREE.Vector3(),
        type: 'observer',
        owner: 'player_001',
        skills: ['monitoring'],
        isActive: true,
        energy: 80,
        maxEnergy: 100,
        specialization: 'system_monitoring',
        thoughtProcess: []
      }

      const helperAgent: AIAgent = {
        id: 'helper_001',
        position: new THREE.Vector3(),
        targetPosition: new THREE.Vector3(),
        type: 'helper',
        owner: 'player_001',
        skills: ['assistance'],
        isActive: true,
        energy: 90,
        maxEnergy: 100,
        specialization: 'player_support',
        thoughtProcess: []
      }

      expect(resolverAgent.type).toBe('resolver')
      expect(observerAgent.type).toBe('observer')
      expect(helperAgent.type).toBe('helper')
    })

    it('should validate energy constraints', () => {
      const agent: AIAgent = {
        id: 'test_agent',
        position: new THREE.Vector3(),
        targetPosition: new THREE.Vector3(),
        type: 'resolver',
        owner: 'player_001',
        skills: [],
        isActive: true,
        energy: 75,
        maxEnergy: 100,
        specialization: 'test',
        thoughtProcess: []
      }

      expect(agent.energy).toBeLessThanOrEqual(agent.maxEnergy)
      expect(agent.energy).toBeGreaterThanOrEqual(0)
    })
  })

  describe('Player', () => {
    it('should create a valid Player', () => {
      const player: Player = {
        id: 'player_001',
        name: 'TestPlayer1',
        nrnBalance: 1000.0,
        agents: [],
        color: '#FF6B6B',
        reputation: 85
      }

      expect(player.id).toBe('player_001')
      expect(player.name).toBe('TestPlayer1')
      expect(player.nrnBalance).toBe(1000.0)
      expect(player.agents).toHaveLength(0)
      expect(player.color).toBe('#FF6B6B')
      expect(player.reputation).toBe(85)
    })

    it('should handle player with agents', () => {
      const agent: AIAgent = {
        id: 'agent_001',
        position: new THREE.Vector3(),
        targetPosition: new THREE.Vector3(),
        type: 'resolver',
        owner: 'player_001',
        skills: ['debugging'],
        isActive: true,
        energy: 100,
        maxEnergy: 100,
        specialization: 'debugging',
        thoughtProcess: []
      }

      const player: Player = {
        id: 'player_001',
        name: 'TestPlayer1',
        nrnBalance: 1000.0,
        agents: [agent],
        color: '#FF6B6B',
        reputation: 85
      }

      expect(player.agents).toHaveLength(1)
      expect(player.agents[0].owner).toBe(player.id)
    })

    it('should validate NRN balance', () => {
      const player: Player = {
        id: 'player_001',
        name: 'TestPlayer1',
        nrnBalance: 500.0,
        agents: [],
        color: '#FF6B6B',
        reputation: 85
      }

      // Simulate spending
      const spendAmount = 100.0
      if (player.nrnBalance >= spendAmount) {
        player.nrnBalance -= spendAmount
      }

      expect(player.nrnBalance).toBe(400.0)
      expect(player.nrnBalance).toBeGreaterThanOrEqual(0)
    })
  })

  describe('GameState', () => {
    it('should create a valid GameState', () => {
      const gameState: GameState = {
        phase: 'active',
        players: [],
        errorNodes: [],
        skillNodes: [],
        currentPlayerId: 'player_001',
        timeRemaining: 1800,
        networkActivity: 75
      }

      expect(gameState.phase).toBe('active')
      expect(gameState.players).toHaveLength(0)
      expect(gameState.errorNodes).toHaveLength(0)
      expect(gameState.skillNodes).toHaveLength(0)
      expect(gameState.currentPlayerId).toBe('player_001')
      expect(gameState.timeRemaining).toBe(1800)
      expect(gameState.networkActivity).toBe(75)
    })

    it('should handle different game phases', () => {
      const lobbyState: GameState = {
        phase: 'lobby',
        players: [],
        errorNodes: [],
        skillNodes: [],
        currentPlayerId: '',
        timeRemaining: 0,
        networkActivity: 0
      }

      const activeState: GameState = {
        phase: 'active',
        players: [],
        errorNodes: [],
        skillNodes: [],
        currentPlayerId: 'player_001',
        timeRemaining: 1800,
        networkActivity: 50
      }

      const endedState: GameState = {
        phase: 'ended',
        players: [],
        errorNodes: [],
        skillNodes: [],
        currentPlayerId: '',
        timeRemaining: 0,
        networkActivity: 0
      }

      expect(lobbyState.phase).toBe('lobby')
      expect(activeState.phase).toBe('active')
      expect(endedState.phase).toBe('ended')
    })
  })

  describe('Controls', () => {
    it('should have all required control types', () => {
      expect(Controls.forward).toBe('forward')
      expect(Controls.backward).toBe('backward')
      expect(Controls.left).toBe('left')
      expect(Controls.right).toBe('right')
      expect(Controls.up).toBe('up')
      expect(Controls.down).toBe('down')
      expect(Controls.select).toBe('select')
      expect(Controls.deploy).toBe('deploy')
      expect(Controls.pause).toBe('pause')
    })

    it('should be usable as object keys', () => {
      const controlMap = {
        [Controls.forward]: 'Move Forward',
        [Controls.backward]: 'Move Backward',
        [Controls.select]: 'Select Target',
        [Controls.deploy]: 'Deploy Agent'
      }

      expect(controlMap[Controls.forward]).toBe('Move Forward')
      expect(controlMap[Controls.select]).toBe('Select Target')
    })
  })
})
