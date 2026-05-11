import React, { useEffect, useState } from 'react';
import { Brain, Shield, Zap, BarChart3 } from 'lucide-react';
import { graphChainApi, SkillNode } from '../services/api';

export default function Capabilities() {
  const [skills, setSkills] = useState<SkillNode[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function load() {
      try {
        const data = await graphChainApi.getAllSkills();
        setSkills(data);
      } catch { /* use empty */ }
      setLoading(false);
    }
    load();
  }, []);

  // Extract unique capabilities across all skills
  const capabilityMap = new Map<string, { count: number; skills: string[] }>();
  skills.forEach(skill => {
    skill.capabilities?.forEach(cap => {
      const existing = capabilityMap.get(cap) || { count: 0, skills: [] };
      existing.count++;
      existing.skills.push(skill.skill_type);
      capabilityMap.set(cap, existing);
    });
  });

  if (loading) {
    return (
      <div className="max-w-7xl mx-auto px-4 py-8">
        <div className="text-center text-gray-400 py-12">Loading capabilities...</div>
      </div>
    );
  }

  return (
    <div className="max-w-7xl mx-auto px-4 py-8">
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-white mb-2">Capabilities</h1>
        <p className="text-gray-400">Agent capabilities extracted from SkillNode registrations on KNIRVCHAIN.</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {Array.from(capabilityMap.entries()).map(([name, data]) => (
          <div key={name} className="bg-gray-800/50 border border-gray-700/50 rounded-xl p-6 hover:border-blue-500/30 transition-all">
            <div className="flex items-center gap-3 mb-3">
              <div className="w-10 h-10 bg-blue-500/20 rounded-lg flex items-center justify-center">
                <Brain className="w-5 h-5 text-blue-400" />
              </div>
              <h3 className="text-lg font-semibold text-white">{name}</h3>
            </div>
            <div className="flex items-center gap-4 text-sm text-gray-400">
              <span className="flex items-center gap-1"><Zap className="w-3 h-3" /> {data.count} skill{data.count !== 1 ? 's' : ''}</span>
            </div>
            <div className="mt-3 flex flex-wrap gap-1">
              {data.skills.slice(0, 3).map(s => (
                <span key={s} className="text-xs px-2 py-1 bg-gray-700/50 rounded text-gray-300">{s}</span>
              ))}
              {data.skills.length > 3 && (
                <span className="text-xs px-2 py-1 bg-gray-700/30 rounded text-gray-500">+{data.skills.length - 3} more</span>
              )}
            </div>
          </div>
        ))}
      </div>

      {capabilityMap.size === 0 && (
        <div className="text-center text-gray-500 py-12">
          <Brain className="w-12 h-12 mx-auto mb-4 opacity-50" />
          <p>No capabilities registered yet. Skills will populate this view as they register on the chain.</p>
        </div>
      )}
    </div>
  );
}
