import React from 'react';
import { useKnirvana } from '../../lib/stores/useKnirvana';
import { Card, CardContent } from './card';
import { Button } from './button';

export default function GameHUD() {
  const { 
    gameTime,
    nrnBalance,
    skillsLearned,
    errorsResolved,
    agents,
    errorNodes,
    skillNodes,
    gamePhase,
    pauseGame,
    startGame
  } = useKnirvana();

  const formatTime = (time: number) => {
    const minutes = Math.floor(time / 60);
    const seconds = Math.floor(time % 60);
    return `${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
  };

  const getNetworkHealth = () => {
    const totalNodes = errorNodes.length + skillNodes.length;
    const resolvedNodes = errorNodes.filter(n => n.progress >= 1).length;
    return totalNodes > 0 ? Math.round((resolvedNodes / totalNodes) * 100) : 0;
  };

  const getActiveAgentsCount = () => {
    return agents.filter(a => a.status === 'working').length;
  };

  const getAverageEfficiency = () => {
    if (agents.length === 0) return 0;
    const totalEfficiency = agents.reduce((sum, agent) => sum + agent.efficiency, 0);
    return Math.round((totalEfficiency / agents.length) * 100);
  };

  return (
    <div className="fixed top-4 left-1/2 transform -translate-x-1/2 z-20 pointer-events-auto">
      <Card className="bg-black bg-opacity-90 border-cyan-500 glow-cyan">
        <CardContent className="py-2 px-4">
          <div className="flex items-center space-x-6 text-sm">
            {/* Game Status */}
            <div className="flex items-center space-x-2">
              <div className={`w-2 h-2 rounded-full ${
                gamePhase === 'playing' ? 'bg-green-400' : 'bg-yellow-400'
              }`} />
              <span className="text-cyan-300 font-mono">
                {gamePhase === 'playing' ? 'ACTIVE' : 'PAUSED'}
              </span>
            </div>

            {/* Game Timer */}
            <div className="text-cyan-400 font-mono">
              <span className="text-cyan-300">TIME:</span>
              <span className="ml-1 text-white font-bold">{formatTime(gameTime)}</span>
            </div>

            {/* NRN Balance */}
            <div className="text-cyan-400 font-mono">
              <span className="text-cyan-300">NRN:</span>
              <span className="ml-1 text-green-400 font-bold">{nrnBalance.toFixed(1)}</span>
            </div>

            {/* Network Stats */}
            <div className="flex space-x-4 text-xs">
              <div className="text-cyan-300">
                <span className="text-cyan-400">Agents:</span>
                <span className="ml-1 text-blue-400">{getActiveAgentsCount()}/{agents.length}</span>
              </div>
              
              <div className="text-cyan-300">
                <span className="text-cyan-400">Efficiency:</span>
                <span className="ml-1 text-yellow-400">{getAverageEfficiency()}%</span>
              </div>
              
              <div className="text-cyan-300">
                <span className="text-cyan-400">Health:</span>
                <span className={`ml-1 ${getNetworkHealth() > 70 ? 'text-green-400' : 
                  getNetworkHealth() > 40 ? 'text-yellow-400' : 'text-red-400'}`}>
                  {getNetworkHealth()}%
                </span>
              </div>
            </div>

            {/* Control Buttons */}
            <div className="flex space-x-2">
              {gamePhase === 'playing' ? (
                <Button
                  onClick={pauseGame}
                  size="sm"
                  className="bg-yellow-600 hover:bg-yellow-700 text-white text-xs px-2 py-1"
                >
                  PAUSE
                </Button>
              ) : (
                <Button
                  onClick={startGame}
                  size="sm"
                  className="bg-green-600 hover:bg-green-700 text-white text-xs px-2 py-1"
                >
                  RESUME
                </Button>
              )}
            </div>
          </div>

          {/* Secondary stats bar */}
          <div className="flex justify-between items-center mt-2 pt-2 border-t border-cyan-700 text-xs">
            <div className="flex space-x-4">
              <span className="text-red-400">
                <span className="text-red-300">ErrorNodes:</span> {errorNodes.filter(n => !n.isBeingSolved).length}
              </span>
              <span className="text-green-400">
                <span className="text-green-300">SkillNodes:</span> {skillNodes.length}
              </span>
              <span className="text-purple-400">
                <span className="text-purple-300">Resolved:</span> {errorsResolved}
              </span>
            </div>
            
            <div className="text-cyan-400">
              <span className="text-cyan-300">Network:</span> KNIRV D-TEN
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
