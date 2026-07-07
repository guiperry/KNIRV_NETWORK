import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { blockchainApi, Block } from '../services/api';
import { ArrowLeft, Clock, Hash, User, CreditCard, Database, ChevronRight } from 'lucide-react';
import LoadingSpinner from '../components/LoadingSpinner';

const BlockDetails: React.FC = () => {
  const { height } = useParams<{ height: string }>();
  const [block, setBlock] = useState<Block | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchBlock = async () => {
      if (!height) return;

      setLoading(true);
      setError(null);

      try {
        const blockData = await blockchainApi.getBlock(parseInt(height));
        setBlock(blockData);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch block');
      } finally {
        setLoading(false);
      }
    };

    fetchBlock();
  }, [height]);

  const formatTime = (timestamp: string) => {
    return new Date(timestamp).toLocaleString();
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <LoadingSpinner size="large" />
      </div>
    );
  }

  if (error || !block) {
    return (
      <div className="max-w-7xl mx-auto px-4 py-8">
        <div className="bg-red-500/10 border border-red-500/20 rounded-lg p-6 text-center">
          <div className="text-red-400 text-lg font-medium mb-2">Block Not Found</div>
          <div className="text-gray-400 mb-4">{error || 'Block does not exist'}</div>
          <Link
            to="/blocks"
            className="inline-flex items-center space-x-2 px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded-lg transition-colors"
          >
            <ArrowLeft className="w-4 h-4" />
            <span>Back to Blocks</span>
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-7xl mx-auto px-4 py-8">
      {/* Header */}
      <div className="flex items-center space-x-4 mb-8">
        <Link
          to="/blocks"
          className="p-2 bg-gray-700 hover:bg-gray-600 rounded-lg transition-colors"
        >
          <ArrowLeft className="w-5 h-5 text-white" />
        </Link>
        <div>
          <h1 className="text-3xl font-bold text-white">Block #{block.header.height}</h1>
          <p className="text-gray-400">Detailed information about this block</p>
        </div>
      </div>

      {/* Block Overview */}
      <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50 mb-8">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          <div className="text-center">
            <div className="text-2xl font-bold text-blue-400">{block.header.height}</div>
            <div className="text-gray-400">Block Height</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-green-400">{block.transactions.length}</div>
            <div className="text-gray-400">Transactions</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-purple-400">
              {formatTime(block.header.timestamp).split(',')[1]?.trim() || 'N/A'}
            </div>
            <div className="text-gray-400">Time</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-orange-400">
              {(JSON.stringify(block).length / 1024).toFixed(1)}KB
            </div>
            <div className="text-gray-400">Size</div>
          </div>
        </div>
      </div>

      {/* Block Details */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 mb-8">
        {/* Basic Information */}
        <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
          <h2 className="text-xl font-semibold text-white mb-6 flex items-center space-x-2">
            <Database className="w-5 h-5 text-blue-400" />
            <span>Block Information</span>
          </h2>
          
          <div className="space-y-4">
            <div>
              <label className="block text-gray-400 text-sm mb-1">Block Hash</label>
              <div className="flex items-center space-x-2 bg-gray-700/50 rounded-lg p-3">
                <Hash className="w-4 h-4 text-gray-400" />
                <code 
                  className="text-gray-300 font-mono text-sm flex-1 cursor-pointer hover:text-white"
                  onClick={() => copyToClipboard(block.hash)}
                  title="Click to copy"
                >
                  {block.hash}
                </code>
              </div>
            </div>

            <div>
              <label className="block text-gray-400 text-sm mb-1">Previous Hash</label>
              <div className="flex items-center space-x-2 bg-gray-700/50 rounded-lg p-3">
                <Hash className="w-4 h-4 text-gray-400" />
                <code 
                  className="text-gray-300 font-mono text-sm flex-1 cursor-pointer hover:text-white"
                  onClick={() => copyToClipboard(block.header.previous_hash)}
                  title="Click to copy"
                >
                  {block.header.previous_hash}
                </code>
              </div>
            </div>

            <div>
              <label className="block text-gray-400 text-sm mb-1">Merkle Root</label>
              <div className="flex items-center space-x-2 bg-gray-700/50 rounded-lg p-3">
                <Hash className="w-4 h-4 text-gray-400" />
                <code 
                  className="text-gray-300 font-mono text-sm flex-1 cursor-pointer hover:text-white"
                  onClick={() => copyToClipboard(block.header.merkle_root)}
                  title="Click to copy"
                >
                  {block.header.merkle_root}
                </code>
              </div>
            </div>

            <div>
              <label className="block text-gray-400 text-sm mb-1">Timestamp</label>
              <div className="flex items-center space-x-2 bg-gray-700/50 rounded-lg p-3">
                <Clock className="w-4 h-4 text-gray-400" />
                <span className="text-gray-300">{formatTime(block.header.timestamp)}</span>
              </div>
            </div>
          </div>
        </div>

        {/* Consensus Information */}
        <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
          <h2 className="text-xl font-semibold text-white mb-6 flex items-center space-x-2">
            <User className="w-5 h-5 text-purple-400" />
            <span>Consensus Information</span>
          </h2>
          
          <div className="space-y-4">
            <div>
              <label className="block text-gray-400 text-sm mb-1">Proposer</label>
              <div className="flex items-center space-x-2 bg-gray-700/50 rounded-lg p-3">
                <User className="w-4 h-4 text-gray-400" />
                <code 
                  className="text-gray-300 font-mono text-sm flex-1 cursor-pointer hover:text-white"
                  onClick={() => copyToClipboard(block.header.proposer)}
                  title="Click to copy"
                >
                  {block.header.proposer}
                </code>
              </div>
            </div>

            <div>
              <label className="block text-gray-400 text-sm mb-1">State Root</label>
              <div className="flex items-center space-x-2 bg-gray-700/50 rounded-lg p-3">
                <Hash className="w-4 h-4 text-gray-400" />
                <code 
                  className="text-gray-300 font-mono text-sm flex-1 cursor-pointer hover:text-white"
                  onClick={() => copyToClipboard(block.header.state_root)}
                  title="Click to copy"
                >
                  {block.header.state_root}
                </code>
              </div>
            </div>

            <div>
              <label className="block text-gray-400 text-sm mb-1">Validator Set</label>
              <div className="bg-gray-700/50 rounded-lg p-3">
                {block.header.validator_set.length > 0 ? (
                  <div className="space-y-2">
                    {block.header.validator_set.map((validator, index) => (
                      <div key={index} className="flex items-center space-x-2">
                        <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                        <code className="text-gray-300 font-mono text-sm">{validator}</code>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="text-gray-400 text-sm">No validators</div>
                )}
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Transactions */}
      <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
        <h2 className="text-xl font-semibold text-white mb-6 flex items-center space-x-2">
          <CreditCard className="w-5 h-5 text-green-400" />
          <span>Transactions ({block.transactions.length})</span>
        </h2>

        {block.transactions.length > 0 ? (
          <div className="space-y-4">
            {block.transactions.map((tx, index) => (
              <div key={tx.id} className="bg-gray-700/30 rounded-lg p-4 border border-gray-600/50">
                <div className="flex items-center justify-between mb-3">
                  <div className="flex items-center space-x-3">
                    <div className="w-8 h-8 bg-green-500/20 rounded-lg flex items-center justify-center">
                      <span className="text-green-400 font-semibold text-sm">{index + 1}</span>
                    </div>
                    <div>
                      <div className="text-white font-medium">Transaction {tx.id}</div>
                      <div className="text-gray-400 text-sm">{formatTime(tx.timestamp)}</div>
                    </div>
                  </div>
                  <div className="text-right">
                    <div className="text-white font-semibold">{tx.amount} tokens</div>
                    <div className="text-gray-400 text-sm">Fee: {tx.fee}</div>
                  </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
                  <div>
                    <span className="text-gray-400">From:</span>
                    <div className="text-gray-300 font-mono mt-1">{tx.from}</div>
                  </div>
                  <div>
                    <span className="text-gray-400">To:</span>
                    <div className="text-gray-300 font-mono mt-1">{tx.to}</div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="text-center py-8 text-gray-400">
            <CreditCard className="w-12 h-12 mx-auto mb-3 opacity-50" />
            <p>No transactions in this block</p>
          </div>
        )}
      </div>

      {/* Navigation */}
      <div className="flex items-center justify-between mt-8">
        {block.header.height > 1 && (
          <Link
            to={`/block/${block.header.height - 1}`}
            className="flex items-center space-x-2 px-4 py-2 bg-gray-700 hover:bg-gray-600 text-white rounded-lg transition-colors"
          >
            <ArrowLeft className="w-4 h-4" />
            <span>Previous Block</span>
          </Link>
        )}
        
        <div className="flex-1"></div>
        
        <Link
          to={`/block/${block.header.height + 1}`}
          className="flex items-center space-x-2 px-4 py-2 bg-gray-700 hover:bg-gray-600 text-white rounded-lg transition-colors"
        >
          <span>Next Block</span>
          <ChevronRight className="w-4 h-4" />
        </Link>
      </div>
    </div>
  );
};

export default BlockDetails;