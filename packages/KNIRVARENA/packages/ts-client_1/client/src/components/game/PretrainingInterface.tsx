import React, { useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '../ui/card';
import { Button } from '../ui/button';
import { Progress } from '../ui/progress';
import { useKnirvana } from '../../lib/stores/useKnirvana';

interface PretrainingSkill {
  id: string;
  name: string;
  category: 'Analysis' | 'Optimization' | 'Synthesis' | 'Debug';
  description: string;
  nrnCost: number;
  trainingTime: number;
  effectiveness: number;
}

export default function PretrainingInterface() {
  const [isOpen, setIsOpen] = useState(false);
  const [selectedAgent, setSelectedAgent] = useState<string | null>(null);
  const [trainingProgress, setTrainingProgress] = useState<{ [key: string]: number }>({});
  const { agents, nrnBalance, spendNRN } = useKnirvana();

  const availableSkills: PretrainingSkill[] = [
    {
      id: 'pattern-analysis',
      name: 'Advanced Pattern Analysis',
      category: 'Analysis',
      description: 'Improves agent ability to identify complex error patterns in KNIRVGRAPH',
      nrnCost: 35,
      trainingTime: 15,
      effectiveness: 0.25
    },
    {
      id: 'neural-optimization',
      name: 'Neural Path Optimization',
      category: 'Optimization', 
      description: 'Enhances solution path finding and computational efficiency',
      nrnCost: 45,
      trainingTime: 20,
      effectiveness: 0.3
    },
    {
      id: 'knowledge-synthesis',
      name: 'Knowledge Synthesis',
      category: 'Synthesis',
      description: 'Enables creation of new SkillNodes from resolved ErrorNodes',
      nrnCost: 55,
      trainingTime: 25,
      effectiveness: 0.35
    },
    {
      id: 'deep-debugging',
      name: 'Deep System Debugging',
      category: 'Debug',
      description: 'Specialized in resolving high-difficulty system ErrorNodes',
      nrnCost: 65,
      trainingTime: 30,
      effectiveness: 0.4
    }
  ];

  const startTraining = (agentId: string, skillId: string) => {
    const skill = availableSkills.find(s => s.id === skillId);
    if (!skill) return;

    if (spendNRN(skill.nrnCost)) {
      const trainingKey = `${agentId}-${skillId}`;
      setTrainingProgress(prev => ({ ...prev, [trainingKey]: 0 }));

      // Simulate training progress
      const interval = setInterval(() => {
        setTrainingProgress(prev => {
          const currentProgress = prev[trainingKey] || 0;
          const newProgress = currentProgress + (100 / skill.trainingTime);
          
          if (newProgress >= 100) {
            clearInterval(interval);
            // Training completed - would update agent capabilities here
            console.log(`Agent ${agentId} completed training: ${skill.name}`);
            return { ...prev, [trainingKey]: 100 };
          }
          
          return { ...prev, [trainingKey]: newProgress };
        });
      }, 1000);
    }
  };

  if (!isOpen) {
    return (
      <div className="absolute bottom-4 left-4 pointer-events-auto">
        <Button
          onClick={() => setIsOpen(true)}
          className="bg-purple-600 hover:bg-purple-700 text-white"
        >
          Pre-train Agents
        </Button>
      </div>
    );
  }

  return (
    <div 
      className="fixed inset-0 bg-black bg-opacity-70 flex items-center justify-center z-50"
      style={{ pointerEvents: 'all' }}
      onClick={(e) => {
        e.stopPropagation();
        setIsOpen(false);
      }}
    >
      <div 
        className="w-96 max-h-[80vh] overflow-y-auto bg-black bg-opacity-95 border border-purple-500 rounded-lg"
        onClick={(e) => e.stopPropagation()}
      >
        <Card className="bg-transparent border-0">
          <CardHeader>
            <CardTitle className="text-purple-400 flex items-center justify-between">
              Agent Pre-training Center
              <Button
                onClick={(e) => {
                  e.stopPropagation();
                  setIsOpen(false);
                }}
                size="sm"
                className="bg-gray-700 hover:bg-gray-600 text-xl px-3 py-1 hover:bg-red-600"
              >
                ✕
              </Button>
            </CardTitle>
          </CardHeader>
        <CardContent className="space-y-4">
          {/* Agent Selection */}
          <div>
            <h3 className="text-cyan-400 mb-2">Select Agent to Train:</h3>
            <div className="grid grid-cols-2 gap-2">
              {agents.map(agent => (
                <Button
                  key={agent.id}
                  onClick={() => setSelectedAgent(agent.id)}
                  className={`text-xs p-2 ${
                    selectedAgent === agent.id
                      ? 'bg-cyan-600 hover:bg-cyan-700'
                      : 'bg-gray-700 hover:bg-gray-600'
                  }`}
                >
                  {agent.type} Agent
                  <div className="text-xs opacity-80">
                    Eff: {Math.round(agent.efficiency * 100)}%
                  </div>
                </Button>
              ))}
            </div>
          </div>

          {/* Available Skills */}
          {selectedAgent && (
            <div>
              <h3 className="text-cyan-400 mb-2">Available Training Skills:</h3>
              <div className="space-y-3">
                {availableSkills.map(skill => {
                  const trainingKey = `${selectedAgent}-${skill.id}`;
                  const isTraining = trainingProgress[trainingKey] !== undefined && trainingProgress[trainingKey] < 100;
                  const isCompleted = trainingProgress[trainingKey] >= 100;

                  return (
                    <div
                      key={skill.id}
                      className="border border-gray-600 rounded p-3 space-y-2"
                    >
                      <div className="flex items-center justify-between">
                        <h4 className="text-white font-semibold">{skill.name}</h4>
                        <span className={`px-2 py-1 rounded text-xs ${
                          skill.category === 'Analysis' ? 'bg-blue-600' :
                          skill.category === 'Optimization' ? 'bg-green-600' :
                          skill.category === 'Synthesis' ? 'bg-purple-600' :
                          'bg-red-600'
                        }`}>
                          {skill.category}
                        </span>
                      </div>
                      
                      <p className="text-gray-300 text-sm">{skill.description}</p>
                      
                      <div className="flex items-center justify-between text-xs">
                        <span className="text-green-400">Cost: {skill.nrnCost} NRN</span>
                        <span className="text-yellow-400">Time: {skill.trainingTime}s</span>
                        <span className="text-purple-400">Boost: +{Math.round(skill.effectiveness * 100)}%</span>
                      </div>

                      {isTraining && (
                        <div className="space-y-1">
                          <div className="flex justify-between text-xs">
                            <span className="text-cyan-400">Training Progress:</span>
                            <span className="text-white">{Math.round(trainingProgress[trainingKey])}%</span>
                          </div>
                          <Progress 
                            value={trainingProgress[trainingKey]} 
                            className="h-2 bg-gray-700"
                          />
                        </div>
                      )}

                      <Button
                        onClick={() => startTraining(selectedAgent, skill.id)}
                        disabled={nrnBalance < skill.nrnCost || isTraining || isCompleted}
                        className={`w-full text-xs ${
                          isCompleted ? 'bg-green-600 hover:bg-green-700' :
                          isTraining ? 'bg-yellow-600 hover:bg-yellow-700' :
                          'bg-purple-600 hover:bg-purple-700'
                        }`}
                      >
                        {isCompleted ? '✓ Training Complete' :
                         isTraining ? 'Training...' :
                         'Start Training'
                        }
                      </Button>
                    </div>
                  );
                })}
              </div>
            </div>
          )}

          {/* NRN Balance */}
          <div className="pt-3 border-t border-gray-700 text-center">
            <div className="text-gray-400 text-sm">Available Balance:</div>
            <div className="text-green-400 font-bold text-lg">{nrnBalance.toFixed(1)} NRN</div>
          </div>
        </CardContent>
      </Card>
      </div>
    </div>
  );
}