import * as THREE from 'three';

export interface ErrorNode {
  id: string;
  position: THREE.Vector3;
  type: 'error';
  difficulty: number;
  bounty: number;
  isBeingResolved: boolean;
  resolvedBy?: string;
  description: string;
  progress: number;
}

export interface SkillNode {
  id: string;
  position: THREE.Vector3;
  type: 'skill';
  name: string;
  category: string;
  createdBy: string;
  usageCount: number;
  value: number;
}

export interface AIAgent {
  id: string;
  position: THREE.Vector3;
  targetPosition: THREE.Vector3;
  type: 'resolver' | 'observer' | 'helper';
  owner: string;
  skills: string[];
  currentTask?: string;
  isActive: boolean;
  energy: number;
  maxEnergy: number;
  specialization: string;
  thoughtProcess: string[];
}

export interface Player {
  id: string;
  name: string;
  nrnBalance: number;
  agents: AIAgent[];
  color: string;
  reputation: number;
}

export interface GameState {
  phase: 'lobby' | 'active' | 'ended';
  players: Player[];
  errorNodes: ErrorNode[];
  skillNodes: SkillNode[];
  currentPlayerId: string;
  timeRemaining: number;
  networkActivity: number;
}

export enum Controls {
  forward = 'forward',
  backward = 'backward',
  left = 'left',
  right = 'right',
  up = 'up',
  down = 'down',
  select = 'select',
  deploy = 'deploy',
  pause = 'pause'
}
