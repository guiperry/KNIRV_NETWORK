import * as THREE from 'three';
import { ErrorNode, SkillNode, AIAgent } from '@/types/game';

export const generateRandomPosition = (radius: number = 50): THREE.Vector3 => {
  const angle = Math.random() * Math.PI * 2;
  const distance = Math.random() * radius;
  return new THREE.Vector3(
    Math.cos(angle) * distance,
    Math.random() * 5 + 1, // Height between 1 and 6
    Math.sin(angle) * distance
  );
};

export const createErrorNode = (id: string): ErrorNode => ({
  id,
  position: generateRandomPosition(40),
  type: 'error',
  difficulty: Math.floor(Math.random() * 5) + 1,
  bounty: Math.floor(Math.random() * 100) + 50,
  isBeingResolved: false,
  description: `Error in ${['Network Routing', 'Agent Processing', 'Knowledge Graph', 'Skill Validation'][Math.floor(Math.random() * 4)]}`,
  progress: 0
});

export const createSkillNode = (id: string, createdBy: string): SkillNode => ({
  id,
  position: generateRandomPosition(30),
  type: 'skill',
  name: `${['Neural', 'Graph', 'Vector', 'Quantum'][Math.floor(Math.random() * 4)]} ${['Resolver', 'Analyzer', 'Optimizer', 'Validator'][Math.floor(Math.random() * 4)]}`,
  category: ['Processing', 'Analysis', 'Validation', 'Optimization'][Math.floor(Math.random() * 4)],
  createdBy,
  usageCount: Math.floor(Math.random() * 50),
  value: Math.floor(Math.random() * 200) + 100
});

export const createAIAgent = (id: string, owner: string): AIAgent => ({
  id,
  position: new THREE.Vector3(Math.random() * 10 - 5, 1, Math.random() * 10 - 5),
  targetPosition: new THREE.Vector3(0, 1, 0),
  type: ['resolver', 'observer', 'helper'][Math.floor(Math.random() * 3)] as 'resolver' | 'observer' | 'helper',
  owner,
  skills: [],
  isActive: true,
  energy: 100,
  maxEnergy: 100,
  specialization: ['Neural Networks', 'Graph Analysis', 'Error Detection', 'Skill Optimization'][Math.floor(Math.random() * 4)],
  thoughtProcess: ['Analyzing target...', 'Computing solution...', 'Optimizing approach...']
});

export const calculateDistance = (pos1: THREE.Vector3, pos2: THREE.Vector3): number => {
  return pos1.distanceTo(pos2);
};

export const moveAgentToTarget = (agent: AIAgent, deltaTime: number, speed: number = 5): void => {
  const direction = new THREE.Vector3().subVectors(agent.targetPosition, agent.position);
  const distance = direction.length();
  
  if (distance > 0.1) {
    direction.normalize();
    agent.position.add(direction.multiplyScalar(speed * deltaTime));
  }
};

export const formatNRN = (value: number): string => {
  return `${value.toLocaleString()} NRN`;
};

export const getNodeColor = (type: 'error' | 'skill', intensity: number = 1): string => {
  if (type === 'error') {
    return `hsl(${Math.sin(Date.now() * 0.003) * 20 + 0}, 100%, ${50 + intensity * 20}%)`;
  }
  return `hsl(${Math.sin(Date.now() * 0.002) * 20 + 180}, 100%, ${50 + intensity * 20}%)`;
};
