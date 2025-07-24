import React, { useEffect, useState } from 'react';
import { blockchainApi, Block } from '../services/api';
import { useBlockchain } from '../context/BlockchainContext';
import { Blocks as BlocksIcon, ChevronLeft, ChevronRight } from 'lucide-react';
import LoadingSpinner from '../components/LoadingSpinner';
import BlockCard from '../components/BlockCard';

const Blocks: React.FC = () => {
  const { currentHeight, isLoading } = useBlockchain();
  const [blocks, setBlocks] = useState<Block[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [error, setError] = useState<string | null>(null);

  const blocksPerPage = 10;
  const totalPages = Math.ceil(currentHeight / blocksPerPage);

  useEffect(() => {
    const fetchBlocks = async () => {
      if (currentHeight === 0) return;

      setLoading(true);
      setError(null);
      
      try {
        const startHeight = Math.max(1, currentHeight - (page - 1) * blocksPerPage - blocksPerPage + 1);
        const endHeight = Math.max(1, currentHeight - (page - 1) * blocksPerPage);
        
        const blockPromises = [];
        for (let height = endHeight; height >= startHeight; height--) {
          blockPromises.push(blockchainApi.getBlock(height));
        }
        
        const blockResults = await Promise.allSettled(blockPromises);
        const fetchedBlocks = blockResults
          .filter((result): result is PromiseFulfilledResult<Block> => result.status === 'fulfilled')
          .map(result => result.value);
        
        setBlocks(fetchedBlocks);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch blocks');
      } finally {
        setLoading(false);
      }
    };

    fetchBlocks();
  }, [currentHeight, page]);

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
        <BlocksIcon className="w-8 h-8 text-blue-400" />
        <div>
          <h1 className="text-3xl font-bold text-white">Blocks</h1>
          <p className="text-gray-400">Browse all blocks in the blockchain</p>
        </div>
      </div>

      {/* Stats */}
      <div className="bg-gray-800/50 backdrop-blur-xl rounded-xl p-6 border border-gray-700/50 mb-8">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <div className="text-center">
            <div className="text-3xl font-bold text-blue-400">{currentHeight}</div>
            <div className="text-gray-400">Total Blocks</div>
          </div>
          <div className="text-center">
            <div className="text-3xl font-bold text-green-400">{page}</div>
            <div className="text-gray-400">Current Page</div>
          </div>
          <div className="text-center">
            <div className="text-3xl font-bold text-purple-400">{totalPages}</div>
            <div className="text-gray-400">Total Pages</div>
          </div>
        </div>
      </div>

      {/* Blocks List */}
      {loading ? (
        <div className="flex items-center justify-center py-12">
          <LoadingSpinner size="large" />
        </div>
      ) : error ? (
        <div className="bg-red-500/10 border border-red-500/20 rounded-lg p-6 text-center">
          <div className="text-red-400 text-lg font-medium mb-2">Error Loading Blocks</div>
          <div className="text-gray-400">{error}</div>
        </div>
      ) : (
        <div className="space-y-4">
          {blocks.map((block) => (
            <BlockCard key={block.hash} block={block} />
          ))}
        </div>
      )}

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-center space-x-4 mt-8">
          <button
            onClick={() => setPage(Math.max(1, page - 1))}
            disabled={page === 1}
            className="flex items-center space-x-2 px-4 py-2 bg-gray-700 hover:bg-gray-600 disabled:bg-gray-800 disabled:text-gray-500 text-white rounded-lg transition-colors"
          >
            <ChevronLeft className="w-4 h-4" />
            <span>Previous</span>
          </button>
          
          <div className="flex items-center space-x-2">
            {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
              const pageNum = Math.max(1, Math.min(totalPages, page - 2 + i));
              return (
                <button
                  key={pageNum}
                  onClick={() => setPage(pageNum)}
                  className={`px-3 py-2 rounded-lg transition-colors ${
                    page === pageNum
                      ? 'bg-blue-500 text-white'
                      : 'bg-gray-700 hover:bg-gray-600 text-gray-300'
                  }`}
                >
                  {pageNum}
                </button>
              );
            })}
          </div>
          
          <button
            onClick={() => setPage(Math.min(totalPages, page + 1))}
            disabled={page === totalPages}
            className="flex items-center space-x-2 px-4 py-2 bg-gray-700 hover:bg-gray-600 disabled:bg-gray-800 disabled:text-gray-500 text-white rounded-lg transition-colors"
          >
            <span>Next</span>
            <ChevronRight className="w-4 h-4" />
          </button>
        </div>
      )}
    </div>
  );
};

export default Blocks;