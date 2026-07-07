import React from 'react';
import { Link } from 'react-router-dom';
import { Block } from '../services/api';
import { Clock, Hash, CreditCard, User } from 'lucide-react';

interface BlockCardProps {
  block: Block;
}

const BlockCard: React.FC<BlockCardProps> = ({ block }) => {
  const formatTime = (timestamp: string) => {
    return new Date(timestamp).toLocaleString();
  };

  const truncateHash = (hash: string, length: number = 12) => {
    return `${hash.slice(0, length)}...${hash.slice(-4)}`;
  };

  return (
    <Link
      to={`/block/${block.header.height}`}
      className="block bg-gray-700/30 hover:bg-gray-700/50 border border-gray-600/50 hover:border-gray-500/50 rounded-lg p-4 transition-all duration-200 hover:scale-[1.02]"
    >
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center space-x-3">
          <div className="w-10 h-10 bg-blue-500/20 rounded-lg flex items-center justify-center">
            <span className="text-blue-400 font-semibold">#{block.header.height}</span>
          </div>
          <div>
            <div className="text-white font-medium">Block {block.header.height}</div>
            <div className="text-gray-400 text-sm flex items-center space-x-1">
              <Clock className="w-3 h-3" />
              <span>{formatTime(block.header.timestamp)}</span>
            </div>
          </div>
        </div>
        <div className="text-right">
          <div className="flex items-center space-x-1 text-gray-400 text-sm">
            <CreditCard className="w-3 h-3" />
            <span>{block.transactions.length} txs</span>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
        <div className="space-y-2">
          <div className="flex items-center space-x-2 text-sm">
            <Hash className="w-3 h-3 text-gray-400" />
            <span className="text-gray-400">Hash:</span>
            <span className="text-gray-300 font-mono">{truncateHash(block.hash)}</span>
          </div>
          <div className="flex items-center space-x-2 text-sm">
            <User className="w-3 h-3 text-gray-400" />
            <span className="text-gray-400">Proposer:</span>
            <span className="text-gray-300 font-mono">{truncateHash(block.header.proposer, 8)}</span>
          </div>
        </div>
        <div className="space-y-2">
          <div className="text-sm">
            <span className="text-gray-400">Previous Hash:</span>
            <div className="text-gray-300 font-mono text-xs mt-1">{truncateHash(block.header.previous_hash)}</div>
          </div>
        </div>
      </div>
    </Link>
  );
};

export default BlockCard;