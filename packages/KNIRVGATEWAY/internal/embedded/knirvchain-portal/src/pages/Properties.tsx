import React, { useEffect, useState } from 'react';
import { Box, Tag, Clock, Shield } from 'lucide-react';
import { graphChainApi, SkillNode } from '../services/api';

export default function Properties() {
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

  // Extract properties: validation status, requirements, performance metrics
  const validated = skills.filter(s => s.validation?.is_validated);
  const withRequirements = skills.filter(s => Object.keys(s.requirements || {}).length > 0);
  const withPerformance = skills.filter(s => s.performance);

  if (loading) {
    return (
      <div className="max-w-7xl mx-auto px-4 py-8">
        <div className="text-center text-gray-400 py-12">Loading properties...</div>
      </div>
    );
  }

  return (
    <div className="max-w-7xl mx-auto px-4 py-8">
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-white mb-2">Properties</h1>
        <p className="text-gray-400">On-chain property records: validation status, requirements, and performance metrics from KNIRVCHAIN.</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
        <div className="bg-gray-800/50 border border-gray-700/50 rounded-xl p-6">
          <div className="flex items-center gap-3 mb-2">
            <Shield className="w-5 h-5 text-green-400" />
            <h3 className="text-lg font-semibold text-white">Validated</h3>
          </div>
          <div className="text-3xl font-bold text-green-400">{validated.length}</div>
          <div className="text-sm text-gray-400 mt-1">Skills with validation signoff</div>
        </div>
        <div className="bg-gray-800/50 border border-gray-700/50 rounded-xl p-6">
          <div className="flex items-center gap-3 mb-2">
            <Tag className="w-5 h-5 text-purple-400" />
            <h3 className="text-lg font-semibold text-white">Requirements</h3>
          </div>
          <div className="text-3xl font-bold text-purple-400">{withRequirements.length}</div>
          <div className="text-sm text-gray-400 mt-1">Skills with requirement constraints</div>
        </div>
        <div className="bg-gray-800/50 border border-gray-700/50 rounded-xl p-6">
          <div className="flex items-center gap-3 mb-2">
            <Clock className="w-5 h-5 text-amber-400" />
            <h3 className="text-lg font-semibold text-white">Performance</h3>
          </div>
          <div className="text-3xl font-bold text-amber-400">{withPerformance.length}</div>
          <div className="text-sm text-gray-400 mt-1">Skills with performance metrics</div>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {skills.map(skill => (
          <div key={skill.id} className="bg-gray-800/50 border border-gray-700/50 rounded-xl p-5">
            <div className="flex items-center justify-between mb-3">
              <h4 className="font-semibold text-white">{skill.skill_type}</h4>
              {skill.validation?.is_validated && (
                <span className="text-xs px-2 py-1 bg-green-500/20 text-green-400 rounded">Validated</span>
              )}
            </div>
            <div className="space-y-2 text-sm">
              {Object.keys(skill.requirements || {}).length > 0 && (
                <div className="flex items-center gap-2 text-gray-400">
                  <Tag className="w-3 h-3" />
                  <span>Requirements: {Object.keys(skill.requirements).join(', ')}</span>
                </div>
              )}
              {skill.performance && (
                <div className="flex items-center gap-2 text-gray-400">
                  <Clock className="w-3 h-3" />
                  <span>Success rate: {((skill.performance.success_rate || 0) * 100).toFixed(0)}% · {skill.performance.total_resolutions} resolutions</span>
                </div>
              )}
              {skill.validation && (
                <div className="flex items-center gap-2 text-gray-400">
                  <Shield className="w-3 h-3" />
                  <span>Score: {skill.validation.validation_score} · {skill.validation.validated_by?.length || 0} peers</span>
                </div>
              )}
            </div>
          </div>
        ))}
      </div>

      {skills.length === 0 && (
        <div className="text-center text-gray-500 py-12">
          <Box className="w-12 h-12 mx-auto mb-4 opacity-50" />
          <p>No properties registered yet. Skill registrations will populate property records on the chain.</p>
        </div>
      )}
    </div>
  );
}
