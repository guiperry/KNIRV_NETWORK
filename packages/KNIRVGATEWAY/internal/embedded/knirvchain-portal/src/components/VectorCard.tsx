import React from 'react';
import { Link } from 'react-router-dom';
import { NRVVector } from '../services/api';
import { Clock, Hash, Target, User } from 'lucide-react';

interface VectorCardProps {
  vector: NRVVector;
}

const VectorCard: React.FC<VectorCardProps> = ({ vector }) => {
  const formatTime = (timestamp: string) => {
    return new Date(timestamp).toLocaleString();
  };

  const truncateHash = (hash: string, length: number = 12) => {
    return `${hash.slice(0, length)}...${hash.slice(-4)}`;
  };

  const formatCoordinates = (coordinates: number[]) => {
    return coordinates.slice(0, 3).map(coord => coord.toFixed(2)).join(', ');
  };

  return (
    <Link
      to={`/vector/${vector.id}`}
      className="block bg-gray-700/30 hover:bg-gray-700/50 border border-gray-600/50 hover:border-gray-500/50 rounded-lg p-4 transition-all duration-200 hover:scale-[1.02]"
    >
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center space-x-3">
          <div className="w-10 h-10 bg-purple-500/20 rounded-lg flex items-center justify-center">
            <Target className="w-5 h-5 text-purple-400" />
          </div>
          <div>
            <div className="text-white font-medium">Vector {vector.id.slice(0, 8)}</div>
            <div className="text-gray-400 text-sm flex items-center space-x-1">
              <Clock className="w-3 h-3" />
              <span>{formatTime(vector.timestamp)}</span>
            </div>
          </div>
        </div>
        <div className="text-right">
          <div className="text-purple-400 text-sm font-medium">
            Confidence: {(vector.confidence * 100).toFixed(1)}%
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
        <div className="space-y-2">
          <div className="flex items-center space-x-2 text-sm">
            <Hash className="w-3 h-3 text-gray-400" />
            <span className="text-gray-400">Target:</span>
            <span className="text-gray-300 font-mono">{truncateHash(vector.target_hash)}</span>
          </div>
          <div className="flex items-center space-x-2 text-sm">
            <User className="w-3 h-3 text-gray-400" />
            <span className="text-gray-400">Source:</span>
            <span className="text-gray-300 font-mono">{truncateHash(vector.source_peer, 8)}</span>
          </div>
        </div>
        <div className="space-y-2">
          <div className="text-sm">
            <span className="text-gray-400">Coordinates:</span>
            <div className="text-gray-300 font-mono text-xs mt-1">[{formatCoordinates(vector.coordinates)}]</div>
          </div>
        </div>
      </div>
    </Link>
  );
};

export default VectorCard;
