import React, { useState } from 'react';
import { X, Trophy, ChevronDown, ChevronUp, Zap, Clock, Code } from 'lucide-react';
import type { EpochResult } from './stores/useKnirvana';

interface EpochResultsPanelProps {
  result: EpochResult;
  epochNumber: number;
  onClose: () => void;
}

function ScoreBar({ value, max = 1 }: { value: number; max?: number }) {
  const pct = Math.round((value / max) * 100);
  const color =
    pct >= 75 ? 'bg-green-500' :
    pct >= 50 ? 'bg-yellow-500' :
    pct >= 25 ? 'bg-orange-500' :
    'bg-red-500';
  return (
    <div className="flex items-center gap-2">
      <div className="flex-1 bg-gray-700 rounded-full h-1.5">
        <div
          className={`${color} h-1.5 rounded-full transition-all duration-500`}
          style={{ width: `${pct}%` }}
        />
      </div>
      <span className="text-xs text-gray-300 w-8 text-right">{pct}%</span>
    </div>
  );
}

interface AgentResultRowProps {
  result: EpochResult['agentResults'][number];
  isWinner: boolean;
  rank: number;
}

function AgentResultRow({ result, isWinner, rank }: AgentResultRowProps) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div
      className={`rounded-lg border p-3 transition-colors ${
        isWinner
          ? 'border-yellow-500/50 bg-yellow-900/20'
          : 'border-gray-600/40 bg-gray-800/60'
      }`}
    >
      {/* Row header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className={`text-sm font-bold w-6 ${isWinner ? 'text-yellow-400' : 'text-gray-500'}`}>
            #{rank}
          </span>
          {isWinner && <Trophy className="w-4 h-4 text-yellow-400" />}
          <div>
            <div className="text-white text-sm font-medium">{result.agentName}</div>
            <div className="text-gray-400 text-xs">{result.personaName}</div>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <div className="text-right">
            <div className={`text-lg font-bold ${isWinner ? 'text-yellow-400' : 'text-gray-300'}`}>
              {Math.round(result.score * 100)}%
            </div>
          </div>
          <button
            onClick={() => setExpanded(v => !v)}
            className="text-gray-400 hover:text-white transition-colors"
          >
            {expanded ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
          </button>
        </div>
      </div>

      <div className="mt-2">
        <ScoreBar value={result.score} />
      </div>

      {/* Expanded details */}
      {expanded && (
        <div className="mt-3 space-y-3 border-t border-gray-700/50 pt-3">
          {/* Chain of thought */}
          {result.proposal.chainOfThought.length > 0 && (
            <div>
              <div className="text-xs text-gray-400 mb-1 flex items-center gap-1">
                <Zap className="w-3 h-3" /> Chain of Thought
              </div>
              <ol className="space-y-1">
                {result.proposal.chainOfThought.map((step, i) => (
                  <li key={i} className="text-xs text-gray-300 flex gap-2">
                    <span className="text-gray-500 flex-shrink-0">{i + 1}.</span>
                    <span>{step}</span>
                  </li>
                ))}
              </ol>
            </div>
          )}

          {/* Proposed solution */}
          <div>
            <div className="text-xs text-gray-400 mb-1 flex items-center gap-1">
              <Code className="w-3 h-3" /> Proposed Fix
            </div>
            <pre className="text-xs text-green-300 bg-black/40 rounded p-2 overflow-x-auto whitespace-pre-wrap max-h-40 overflow-y-auto">
              {result.proposal.solution}
            </pre>
          </div>

          {/* Judge reasoning */}
          {result.reasoning && (
            <div>
              <div className="text-xs text-gray-400 mb-1">Judge Reasoning</div>
              <p className="text-xs text-gray-300 italic">{result.reasoning}</p>
            </div>
          )}

          {/* Latency */}
          <div className="flex items-center gap-1 text-xs text-gray-500">
            <Clock className="w-3 h-3" />
            <span>Response time: {result.proposal.estimatedLatency.toFixed(0)} ms</span>
          </div>
        </div>
      )}
    </div>
  );
}

export default function EpochResultsPanel({ result, epochNumber, onClose }: EpochResultsPanelProps) {
  const ranked = [...result.agentResults].sort((a, b) => b.score - a.score);

  return (
    <div className="fixed inset-0 bg-black/70 z-[60] flex items-center justify-center p-4">
      <div className="bg-gray-900 border border-cyan-500/40 rounded-xl w-full max-w-2xl max-h-[90vh] flex flex-col shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-gray-700/50">
          <div>
            <h2 className="text-cyan-400 font-bold text-lg">Epoch #{epochNumber} Results</h2>
            <p className="text-gray-400 text-sm mt-0.5">
              Challenge: <span className="text-white">{result.challenge.title}</span>
              <span className="ml-2 text-xs px-1.5 py-0.5 bg-gray-700 rounded text-gray-300">
                {result.challenge.type}
              </span>
            </p>
          </div>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-white transition-colors p-1"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Challenge description */}
        <div className="px-4 pt-3 pb-2">
          <div className="bg-gray-800/60 rounded-lg p-3 text-sm text-gray-300 border border-gray-700/40">
            <span className="text-gray-500 text-xs uppercase tracking-wide block mb-1">Bug Description</span>
            {result.challenge.description}
          </div>
        </div>

        {/* Winner callout */}
        {result.winnerId && (
          <div className="px-4 py-2">
            <div className="bg-yellow-900/30 border border-yellow-500/40 rounded-lg px-4 py-2 flex items-center gap-3">
              <Trophy className="w-5 h-5 text-yellow-400 flex-shrink-0" />
              <div className="text-sm">
                <span className="text-yellow-300 font-semibold">
                  {ranked[0]?.agentName}
                </span>
                <span className="text-gray-300 ml-2">
                  won with a score of{' '}
                  <span className="text-yellow-400 font-bold">
                    {Math.round((result.winnerScore) * 100)}%
                  </span>
                </span>
                {result.winnerScore > 0.8 && (
                  <span className="ml-2 text-yellow-500 text-xs">— Skill Slot Hijacked!</span>
                )}
              </div>
            </div>
          </div>
        )}

        {/* Agent results list */}
        <div className="flex-1 overflow-y-auto px-4 pb-4 space-y-2 mt-2">
          {ranked.map((r, i) => (
            <AgentResultRow
              key={r.agentId}
              result={r}
              isWinner={r.agentId === result.winnerId}
              rank={i + 1}
            />
          ))}
        </div>

        {/* Footer */}
        <div className="border-t border-gray-700/50 p-3 flex justify-end">
          <button
            onClick={onClose}
            className="px-4 py-2 bg-cyan-700 hover:bg-cyan-600 text-white rounded-lg text-sm transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
