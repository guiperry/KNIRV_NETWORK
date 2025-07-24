import React, { useEffect, useState } from 'react';
import { blockchainApi, Block, Transaction } from '../services/api';
import { useBlockchain } from '../context/BlockchainContext';
import { CreditCard, Clock, User, Hash, ArrowRight } from 'lucide-react';
import LoadingSpinner from '../components/LoadingSpinner';

const Transactions: React.FC = () => {
  const { currentHeight, isLoading } = useBlockchain();
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchTransactions = async () => {
      if (currentHeight === 0) return;

      setLoading(true);
      setError(null);

      try {
        // Fetch recent blocks and extract all transactions
        const recentBlocks = await blockchainApi.getRecentBlocks(20);
        const allTransactions: Transaction[] = [];
        
        recentBlocks.forEach(block => {
          allTransactions.push(...block.transactions);
        });

        // Sort by timestamp (most recent first)
        allTransactions.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());
        
        setTransactions(allTransactions);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch transactions');
      } finally {
        setLoading(false);
      }
    };

    fetchTransactions();
  }, [currentHeight]);

  const formatTime = (timestamp: string) => {
    return new Date(timestamp).toLocaleString();
  };

  const truncateHash = (hash: string, length: number = 12) => {
    return `${hash.slice(0, length)}...${hash.slice(-4)}`;
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <LoadingSpinner size="large" />
      </div>
    );
  }

  return (
    <div className="max-w-7xl mx-auto px-4 py-8">
      {/* Header */}
      <div className="flex items-center space-x-3 mb-8">
        <CreditCard className="w-8 h-8 text-green-400" />
        <div>
          <h1 className="text-3xl font-bold text-white">Transactions</h1>
          <p className="text-gray-400">Recent blockchain transactions</p>
        </div>
      </div>

      {/* Stats */}
      <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50 mb-8">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <div className="text-center">
            <div className="text-3xl font-bold text-green-400">{transactions.length}</div>
            <div className="text-gray-400">Total Transactions</div>
          </div>
          <div className="text-center">
            <div className="text-3xl font-bold text-blue-400">
              {transactions.reduce((sum, tx) => sum + tx.amount, 0).toLocaleString()}
            </div>
            <div className="text-gray-400">Total Volume</div>
          </div>
          <div className="text-center">
            <div className="text-3xl font-bold text-purple-400">
              {transactions.reduce((sum, tx) => sum + tx.fee, 0).toLocaleString()}
            </div>
            <div className="text-gray-400">Total Fees</div>
          </div>
        </div>
      </div>

      {/* Transactions List */}
      {loading ? (
        <div className="flex items-center justify-center py-12">
          <LoadingSpinner size="large" />
        </div>
      ) : error ? (
        <div className="bg-red-500/10 border border-red-500/20 rounded-lg p-6 text-center">
          <div className="text-red-400 text-lg font-medium mb-2">Error Loading Transactions</div>
          <div className="text-gray-400">{error}</div>
        </div>
      ) : transactions.length === 0 ? (
        <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-12 border border-gray-700/50 text-center">
          <CreditCard className="w-16 h-16 mx-auto mb-4 text-gray-500" />
          <h3 className="text-xl font-semibold text-gray-400 mb-2">No Transactions Found</h3>
          <p className="text-gray-500">There are no transactions in the recent blocks.</p>
        </div>
      ) : (
        <div className="space-y-4">
          {transactions.map((tx) => (
            <div
              key={tx.id}
              className="bg-gray-800/50 backdrop-blur-xl border border-gray-700/50 rounded-xl p-6 hover:border-gray-600/50 transition-all duration-200"
            >
              <div className="flex items-center justify-between mb-4">
                <div className="flex items-center space-x-3">
                  <div className="w-10 h-10 bg-green-500/20 rounded-lg flex items-center justify-center">
                    <CreditCard className="w-5 h-5 text-green-400" />
                  </div>
                  <div>
                    <div className="text-white font-medium">Transaction</div>
                    <div className="text-gray-400 text-sm flex items-center space-x-1">
                      <Clock className="w-3 h-3" />
                      <span>{formatTime(tx.timestamp)}</span>
                    </div>
                  </div>
                </div>
                <div className="text-right">
                  <div className="text-white font-semibold text-lg">{tx.amount.toLocaleString()} tokens</div>
                  <div className="text-gray-400 text-sm">Fee: {tx.fee} tokens</div>
                </div>
              </div>

              <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 mb-4">
                <div className="space-y-3">
                  <div>
                    <div className="flex items-center space-x-2 text-sm text-gray-400 mb-1">
                      <Hash className="w-3 h-3" />
                      <span>Transaction ID</span>
                    </div>
                    <div className="text-gray-300 font-mono text-sm bg-gray-700/30 rounded p-2">
                      {tx.id}
                    </div>
                  </div>
                  <div>
                    <div className="flex items-center space-x-2 text-sm text-gray-400 mb-1">
                      <User className="w-3 h-3" />
                      <span>From</span>
                    </div>
                    <div className="text-gray-300 font-mono text-sm bg-gray-700/30 rounded p-2">
                      {tx.from}
                    </div>
                  </div>
                </div>
                <div className="space-y-3">
                  <div>
                    <div className="flex items-center space-x-2 text-sm text-gray-400 mb-1">
                      <span>Nonce</span>
                    </div>
                    <div className="text-gray-300 font-mono text-sm bg-gray-700/30 rounded p-2">
                      {tx.nonce}
                    </div>
                  </div>
                  <div>
                    <div className="flex items-center space-x-2 text-sm text-gray-400 mb-1">
                      <User className="w-3 h-3" />
                      <span>To</span>
                    </div>
                    <div className="text-gray-300 font-mono text-sm bg-gray-700/30 rounded p-2">
                      {tx.to}
                    </div>
                  </div>
                </div>
              </div>

              {tx.data && (
                <div className="border-t border-gray-700/50 pt-4">
                  <div className="text-sm text-gray-400 mb-1">Transaction Data</div>
                  <div className="text-gray-300 font-mono text-sm bg-gray-700/30 rounded p-2 max-h-20 overflow-y-auto">
                    {tx.data}
                  </div>
                </div>
              )}

              <div className="flex items-center justify-between mt-4 pt-4 border-t border-gray-700/50">
                <div className="text-sm text-gray-400">
                  Signature: {truncateHash(tx.signature, 16)}
                </div>
                <div className="flex items-center space-x-2 text-sm text-blue-400 hover:text-blue-300">
                  <span>View Details</span>
                  <ArrowRight className="w-3 h-3" />
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default Transactions;