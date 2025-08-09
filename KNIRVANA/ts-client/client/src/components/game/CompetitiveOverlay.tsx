import React from 'react';
import { useKnirvana } from '../../lib/stores/useKnirvana';
import { Card, CardContent } from '../ui/card';

interface CompetitorData {
  id: string;
  name: string;
  nrnBalance: number;
  errorsResolved: number;
  agentsDeployed: number;
  rank: number;
}

export default function CompetitiveOverlay() {
  const { nrnBalance, errorsResolved, agents } = useKnirvana();

  // Mock competitive data - in real implementation this would come from multiplayer system
  const competitors: CompetitorData[] = [
    {
      id: 'player-1',
      name: 'You',
      nrnBalance: nrnBalance,
      errorsResolved: errorsResolved,
      agentsDeployed: agents.length,
      rank: 1
    },
    {
      id: 'player-2',
      name: 'AIExplorer_42',
      nrnBalance: 245.7,
      errorsResolved: 8,
      agentsDeployed: 4,
      rank: 2
    },
    {
      id: 'player-3',
      name: 'GraphMaster',
      nrnBalance: 189.3,
      errorsResolved: 6,
      agentsDeployed: 3,
      rank: 3
    },
    {
      id: 'player-4',
      name: 'NodeHunter',
      nrnBalance: 156.8,
      errorsResolved: 5,
      agentsDeployed: 3,
      rank: 4
    }
  ];

  // Sort by NRN balance to determine current leaderboard
  const sortedCompetitors = competitors.sort((a, b) => b.nrnBalance - a.nrnBalance);

  return (
    <div className="absolute top-4 right-4 w-80 pointer-events-auto">
      <Card className="bg-black bg-opacity-90 border-purple-500 glow-purple">
        <CardContent className="py-3 px-4">
          <div className="text-center mb-3">
            <h3 className="text-purple-400 font-bold text-lg">Live Leaderboard</h3>
            <p className="text-cyan-300 text-xs">KNIRV D-TEN Session</p>
          </div>
          
          <div className="space-y-2">
            {sortedCompetitors.map((competitor, index) => {
              const currentCompetitor = competitor;
              return (
              <div
                key={competitor.id}
                className={`flex items-center justify-between p-2 rounded ${
                  competitor.name === 'You' 
                    ? 'bg-cyan-900 bg-opacity-50 border border-cyan-500' 
                    : 'bg-gray-800 bg-opacity-50'
                }`}
              >
                <div className="flex items-center space-x-2">
                  <div className={`w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold ${
                    index === 0 ? 'bg-yellow-500 text-black' :
                    index === 1 ? 'bg-gray-400 text-black' :
                    index === 2 ? 'bg-orange-600 text-white' :
                    'bg-gray-600 text-white'
                  }`}>
                    {index + 1}
                  </div>
                  <div>
                    <div className={`font-semibold ${
                      competitor.name === 'You' ? 'text-cyan-300' : 'text-white'
                    }`}>
                      {competitor.name}
                    </div>
                    <div className="text-xs text-gray-400">
                      {competitor.errorsResolved} solved • {competitor.agentsDeployed} agents
                    </div>
                  </div>
                </div>
                
                <div className="text-right">
                  <div className="text-green-400 font-bold">
                    {competitor.nrnBalance.toFixed(1)} NRN
                  </div>
                  <div className="text-xs text-gray-400">
                    {competitor.nrnBalance > sortedCompetitors[index + 1]?.nrnBalance 
                      ? `+${(competitor.nrnBalance - (sortedCompetitors[index + 1]?.nrnBalance || 0)).toFixed(1)}`
                      : ''
                    }
                  </div>
                </div>
              </div>
            );
            })}
          </div>
          
          {/* Session stats */}
          <div className="mt-4 pt-3 border-t border-purple-700 grid grid-cols-2 gap-4 text-xs">
            <div className="text-center">
              <div className="text-purple-400">Session Time</div>
              <div className="text-white font-mono">08:42</div>
            </div>
            <div className="text-center">
              <div className="text-purple-400">Active Players</div>
              <div className="text-white font-mono">{competitors.length}</div>
            </div>
            <div className="text-center">
              <div className="text-purple-400">Network Errors</div>
              <div className="text-red-400 font-mono">127</div>
            </div>
            <div className="text-center">
              <div className="text-purple-400">Resolution Rate</div>
              <div className="text-green-400 font-mono">73%</div>
            </div>
          </div>
          
          {/* Competition status */}
          <div className="mt-3 text-center">
            {sortedCompetitors[0].name === 'You' ? (
              <div className="text-yellow-400 text-sm font-semibold animate-pulse">
                🏆 Leading the session!
              </div>
            ) : (
              <div className="text-cyan-400 text-sm">
                Need {(sortedCompetitors[0].nrnBalance - nrnBalance).toFixed(1)} more NRN to lead
              </div>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}