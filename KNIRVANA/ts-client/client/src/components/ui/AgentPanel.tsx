import { useKnirvana } from "../../lib/stores/useKnirvana";
import { Button } from "./button";
import { Card, CardContent, CardHeader, CardTitle } from "./card";
import AgentProfileCard from "./AgentProfileCard";

export default function AgentPanel() {
  const { agents, selectedAgent, selectAgent, nrnBalance, createAgent } = useKnirvana();

  const agentTypes = ['Analyzer', 'Optimizer', 'Synthesizer', 'Debugger'];

  return (
    <div className="absolute top-20 right-4 w-80 space-y-4 pointer-events-auto">
      <Card className="bg-black bg-opacity-80 border-cyan-500">
        <CardHeader>
          <CardTitle className="text-cyan-400 text-lg">AI Agents</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {/* Agent List */}
          <div className="space-y-2 max-h-48 overflow-y-auto">
            {agents.map((agent) => (
              <div
                key={agent.id}
                onClick={() => selectAgent(agent.id)}
                className={`p-2 rounded border cursor-pointer transition-colors ${
                  selectedAgent === agent.id
                    ? 'border-cyan-400 bg-cyan-900 bg-opacity-30'
                    : 'border-gray-600 hover:border-cyan-600'
                }`}
              >
                <div className="flex justify-between items-center">
                  <div>
                    <p className="text-cyan-300 text-sm font-medium">{agent.type}</p>
                    <p className="text-gray-400 text-xs">{agent.status}</p>
                  </div>
                  <div className="text-right">
                    <p className="text-green-400 text-xs">
                      {Math.round(agent.efficiency * 100)}%
                    </p>
                    <div className={`w-2 h-2 rounded-full ${
                      agent.status === 'working' ? 'bg-yellow-400' :
                      agent.status === 'moving' ? 'bg-blue-400' :
                      'bg-gray-400'
                    }`} />
                  </div>
                </div>
              </div>
            ))}
          </div>
          
          {/* Create New Agent */}
          <div className="border-t border-gray-600 pt-3">
            <p className="text-cyan-300 text-sm mb-2">Deploy New Agent:</p>
            <div className="grid grid-cols-2 gap-2">
              {agentTypes.map((type) => (
                <Button
                  key={type}
                  onClick={() => createAgent(type)}
                  disabled={nrnBalance < 50}
                  size="sm"
                  className="bg-gray-700 hover:bg-gray-600 text-xs"
                >
                  {type}
                  <span className="block text-xs text-gray-400">50 NRN</span>
                </Button>
              ))}
            </div>
          </div>
          
          {/* Agent Stats */}
          <div className="border-t border-gray-600 pt-3 text-xs text-gray-400">
            <div className="flex justify-between">
              <span>Total Agents:</span>
              <span className="text-cyan-400">{agents.length}</span>
            </div>
            <div className="flex justify-between">
              <span>Active:</span>
              <span className="text-green-400">
                {agents.filter(a => a.status === 'working').length}
              </span>
            </div>
            <div className="flex justify-between">
              <span>Avg Efficiency:</span>
              <span className="text-yellow-400">
                {agents.length > 0 
                  ? Math.round(agents.reduce((sum, a) => sum + a.efficiency, 0) / agents.length * 100)
                  : 0}%
              </span>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
