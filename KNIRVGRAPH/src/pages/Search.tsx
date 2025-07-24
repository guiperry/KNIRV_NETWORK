import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { useBlockchain } from '../context/BlockchainContext';
import { blockchainApi, Block, Account } from '../services/api';
import { Search as SearchIcon, Blocks, Users, CreditCard, ArrowRight } from 'lucide-react';
import LoadingSpinner from '../components/LoadingSpinner';

const Search: React.FC = () => {
  const { query } = useParams<{ query: string }>();
  const { currentHeight } = useBlockchain();
  const [loading, setLoading] = useState(true);
  const [results, setResults] = useState<{
    block?: Block;
    account?: Account;
    type: 'block' | 'account' | 'transaction' | 'unknown';
  } | null>(null);

  useEffect(() => {
    const performSearch = async () => {
      if (!query) return;

      setLoading(true);
      setResults(null);

      try {
        // Try to determine what type of search this is
        const isNumeric = /^\d+$/.test(query);
        const isAddress = query.startsWith('0x') && query.length === 42;
        const isTxHash = query.startsWith('0x') && query.length === 66;

        if (isNumeric) {
          // Search for block by height
          try {
            const blockHeight = parseInt(query);
            if (blockHeight <= currentHeight) {
              const block = await blockchainApi.getBlock(blockHeight);
              setResults({ block, type: 'block' });
            } else {
              setResults({ type: 'unknown' });
            }
          } catch (error) {
            setResults({ type: 'unknown' });
          }
        } else if (isAddress) {
          // Search for account
          try {
            const account = await blockchainApi.getAccount(query);
            setResults({ account, type: 'account' });
          } catch (error) {
            setResults({ type: 'unknown' });
          }
        } else if (isTxHash) {
          // Transaction hash search (placeholder)
          setResults({ type: 'transaction' });
        } else {
          setResults({ type: 'unknown' });
        }
      } catch (error) {
        setResults({ type: 'unknown' });
      } finally {
        setLoading(false);
      }
    };

    performSearch();
  }, [query, currentHeight]);

  const formatTime = (timestamp: string) => {
    return new Date(timestamp).toLocaleString();
  };

  if (loading) {
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
        <SearchIcon className="w-8 h-8 text-blue-400" />
        <div>
          <h1 className="text-3xl font-bold text-white">Search Results</h1>
          <p className="text-gray-400">Results for: <span className="font-mono text-gray-300">"{query}"</span></p>
        </div>
      </div>

      {/* Results */}
      {!results || results.type === 'unknown' ? (
        <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-12 border border-gray-700/50 text-center">
          <SearchIcon className="w-16 h-16 mx-auto mb-4 text-gray-500" />
          <h3 className="text-xl font-semibold text-gray-400 mb-2">No Results Found</h3>
          <p className="text-gray-500 mb-6">
            We couldn't find any blocks, accounts, or transactions matching your search.
          </p>
          <div className="bg-blue-500/10 border border-blue-500/20 rounded-lg p-4 text-left max-w-md mx-auto">
            <h4 className="text-blue-400 font-medium mb-2">Search Tips:</h4>
            <ul className="text-sm text-gray-300 space-y-1">
              <li>• Block height: Enter a number (e.g., 123)</li>
              <li>• Account address: Start with 0x and 42 characters</li>
              <li>• Transaction hash: Start with 0x and 66 characters</li>
            </ul>
          </div>
        </div>
      ) : results.type === 'block' && results.block ? (
        <div className="space-y-6">
          <div className="bg-blue-500/10 border border-blue-500/20 rounded-lg p-4 flex items-center space-x-3">
            <Blocks className="w-6 h-6 text-blue-400" />
            <div>
              <div className="text-blue-400 font-medium">Block Found</div>
              <div className="text-gray-400 text-sm">Found block #{results.block.header.height}</div>
            </div>
          </div>

          <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-xl font-semibold text-white">Block #{results.block.header.height}</h2>
              <Link
                to={`/block/${results.block.header.height}`}
                className="flex items-center space-x-2 px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded-lg transition-colors"
              >
                <span>View Details</span>
                <ArrowRight className="w-4 h-4" />
              </Link>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
              <div className="text-center">
                <div className="text-2xl font-bold text-blue-400">{results.block.header.height}</div>
                <div className="text-gray-400">Height</div>
              </div>
              <div className="text-center">
                <div className="text-2xl font-bold text-green-400">{results.block.transactions.length}</div>
                <div className="text-gray-400">Transactions</div>
              </div>
              <div className="text-center">
                <div className="text-2xl font-bold text-purple-400">
                  {formatTime(results.block.header.timestamp).split(',')[1]?.trim() || 'N/A'}
                </div>
                <div className="text-gray-400">Time</div>
              </div>
              <div className="text-center">
                <div className="text-2xl font-bold text-orange-400">
                  {(JSON.stringify(results.block).length / 1024).toFixed(1)}KB
                </div>
                <div className="text-gray-400">Size</div>
              </div>
            </div>

            <div className="mt-6 grid grid-cols-1 lg:grid-cols-2 gap-6">
              <div>
                <label className="block text-gray-400 text-sm mb-1">Block Hash</label>
                <div className="bg-gray-700/50 rounded-lg p-3">
                  <code className="text-gray-300 font-mono text-sm break-all">{results.block.hash}</code>
                </div>
              </div>
              <div>
                <label className="block text-gray-400 text-sm mb-1">Previous Hash</label>
                <div className="bg-gray-700/50 rounded-lg p-3">
                  <code className="text-gray-300 font-mono text-sm break-all">{results.block.header.previous_hash}</code>
                </div>
              </div>
            </div>
          </div>
        </div>
      ) : results.type === 'account' && results.account ? (
        <div className="space-y-6">
          <div className="bg-purple-500/10 border border-purple-500/20 rounded-lg p-4 flex items-center space-x-3">
            <Users className="w-6 h-6 text-purple-400" />
            <div>
              <div className="text-purple-400 font-medium">Account Found</div>
              <div className="text-gray-400 text-sm">Found account with balance {results.account.balance.toLocaleString()} tokens</div>
            </div>
          </div>

          <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50">
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-xl font-semibold text-white">Account Details</h2>
              <Link
                to={`/account/${results.account.address}`}
                className="flex items-center space-x-2 px-4 py-2 bg-purple-500 hover:bg-purple-600 text-white rounded-lg transition-colors"
              >
                <span>View Details</span>
                <ArrowRight className="w-4 h-4" />
              </Link>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
              <div className="text-center">
                <div className="text-2xl font-bold text-blue-400">{results.account.balance.toLocaleString()}</div>
                <div className="text-gray-400">Balance</div>
              </div>
              <div className="text-center">
                <div className="text-2xl font-bold text-green-400">{results.account.nonce}</div>
                <div className="text-gray-400">Nonce</div>
              </div>
              <div className="text-center">
                <div className="text-2xl font-bold text-purple-400">Regular</div>
                <div className="text-gray-400">Type</div>
              </div>
            </div>

            <div className="mt-6">
              <label className="block text-gray-400 text-sm mb-1">Account Address</label>
              <div className="bg-gray-700/50 rounded-lg p-3">
                <code className="text-gray-300 font-mono text-sm break-all">{results.account.address}</code>
              </div>
            </div>
          </div>
        </div>
      ) : results.type === 'transaction' ? (
        <div className="space-y-6">
          <div className="bg-green-500/10 border border-green-500/20 rounded-lg p-4 flex items-center space-x-3">
            <CreditCard className="w-6 h-6 text-green-400" />
            <div>
              <div className="text-green-400 font-medium">Transaction Hash</div>
              <div className="text-gray-400 text-sm">Transaction search functionality coming soon</div>
            </div>
          </div>

          <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-12 border border-gray-700/50 text-center">
            <CreditCard className="w-16 h-16 mx-auto mb-4 text-gray-500" />
            <h3 className="text-xl font-semibold text-gray-400 mb-2">Transaction Search</h3>
            <p className="text-gray-500">
              Transaction hash search is not yet implemented. Please search for blocks or accounts instead.
            </p>
          </div>
        </div>
      ) : null}

      {/* Search Suggestions */}
      <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50 mt-8">
        <h3 className="text-lg font-semibold text-white mb-4">Try These Searches</h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <Link
            to="/search/1"
            className="group bg-blue-500/10 border border-blue-500/20 rounded-lg p-4 hover:border-blue-500/40 transition-all"
          >
            <Blocks className="w-6 h-6 text-blue-400 mb-2" />
            <div className="text-white font-medium">Block #1</div>
            <div className="text-gray-400 text-sm">Genesis block</div>
          </Link>

          <Link
            to="/search/0x1234567890123456789012345678901234567890"
            className="group bg-purple-500/10 border border-purple-500/20 rounded-lg p-4 hover:border-purple-500/40 transition-all"
          >
            <Users className="w-6 h-6 text-purple-400 mb-2" />
            <div className="text-white font-medium">Sample Account</div>
            <div className="text-gray-400 text-sm">View account details</div>
          </Link>

          <Link
            to={`/search/${currentHeight}`}
            className="group bg-green-500/10 border border-green-500/20 rounded-lg p-4 hover:border-green-500/40 transition-all"
          >
            <Blocks className="w-6 h-6 text-green-400 mb-2" />
            <div className="text-white font-medium">Latest Block</div>
            <div className="text-gray-400 text-sm">Block #{currentHeight}</div>
          </Link>
        </div>
      </div>
    </div>
  );
};

export default Search;